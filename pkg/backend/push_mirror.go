package backend

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/ssrf"
)

// AddPushMirror adds a push mirror to a repository.
func (b *Backend) AddPushMirror(ctx context.Context, repo proto.Repository, name, remoteURL string) error {
	return b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return b.store.CreatePushMirror(ctx, tx, repo.ID(), name, remoteURL)
	})
}

// RemovePushMirror removes a push mirror from a repository.
func (b *Backend) RemovePushMirror(ctx context.Context, repo proto.Repository, name string) error {
	return b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return b.store.DeletePushMirror(ctx, tx, repo.ID(), name)
	})
}

// ListPushMirrors lists all push mirrors for a repository.
func (b *Backend) ListPushMirrors(ctx context.Context, repo proto.Repository) ([]models.PushMirror, error) {
	var mirrors []models.PushMirror
	err := b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		mirrors, err = b.store.GetPushMirrorsByRepoID(ctx, tx, repo.ID())
		return err
	})
	return mirrors, err
}

// PushMirrors triggers an async push to all enabled mirrors for a repository.
// Called from the post-receive hook. Errors are logged but not fatal.
func (b *Backend) PushMirrors(ctx context.Context, repo proto.Repository) {
	mirrors, err := b.ListPushMirrors(ctx, repo)
	if err != nil {
		b.logger.Warn("push-mirror: failed to list mirrors", "repo", repo.Name(), "err", err)
		return
	}
	repoPath := b.repoPath(repo.Name())
	const mirrorPushTimeout = 10 * time.Minute
	const maxConcurrentPushes = 5
	sem := make(chan struct{}, maxConcurrentPushes)
	var wg sync.WaitGroup
	for _, m := range mirrors {
		if !m.Enabled {
			continue
		}
		u, err := url.Parse(m.RemoteURL)
		if err == nil && (u.Scheme == "http" || u.Scheme == "https") {
			if ssrfErr := ssrf.ValidateURL(ctx, m.RemoteURL); ssrfErr != nil {
				b.logger.Warn("push mirror: SSRF check failed", "remote", m.RemoteURL, "err", ssrfErr)
				continue
			}
		} else if err == nil && (u.Scheme == "ssh" || u.Scheme == "git+ssh" || u.Scheme == "ssh+git") {
			// Validate ssh:// / git+ssh:// / ssh+git:// scheme URLs against SSRF.
			// Note: unlike the HTTP client (NewSecureClient), the SSH mirror uses
			// the git subprocess which re-resolves the hostname at dial time —
			// there is a DNS rebinding window between this check and the actual
			// connection. Mitigate by ensuring the host resolves to a public IP.
			if ssrfErr := ssrf.ValidateHost(ctx, u.Hostname()); ssrfErr != nil {
				b.logger.Warn("push mirror: SSRF check failed", "remote", m.RemoteURL, "err", ssrfErr)
				continue
			}
		} else if err != nil || u.Scheme == "" {
			// SCP-style remote (e.g. git@host:repo) — url.Parse either fails or
			// produces an empty scheme. Both cases require manual host extraction.
			host, scpErr := extractSCPHost(m.RemoteURL)
			if scpErr != nil {
				b.logger.Warn("push mirror: cannot extract host from SCP-style remote", "remote", m.RemoteURL, "err", scpErr)
				continue
			}
			if ssrfErr := ssrf.ValidateHost(ctx, host); ssrfErr != nil {
				b.logger.Warn("push mirror: SSRF check failed", "remote", m.RemoteURL, "err", ssrfErr)
				continue
			}
		} else {
			// Block git://, file://, and any other unrecognized scheme.
			// file:// could allow access to local filesystem paths and is
			// blocked here even though it does not carry a network host.
			schemeErr := fmt.Errorf("push mirror: unsupported URL scheme %q", u.Scheme)
			b.logger.Warn(schemeErr.Error(), "remote", m.RemoteURL)
			continue
		}
		sem <- struct{}{} // acquire
		wg.Add(1)
		go func(m models.PushMirror) {
			defer wg.Done()
			defer func() { <-sem }() // release
			mirrorCtx, cancel := context.WithTimeout(ctx, mirrorPushTimeout)
			defer cancel()
			// m.RemoteURL is passed as a positional argument to exec.Command (not shell-expanded),
			// so there is no shell injection risk. SSRF validation has already run above.
			cmd := exec.CommandContext(mirrorCtx, "git", "push", "--mirror", m.RemoteURL)
			cmd.Dir = repoPath
			// Note: HOME and PATH are sourced from the current process environment.
			// If soft-serve runs as a system service with a restricted environment,
			// these may be empty; in that case, git push may fail. Ensure the service
			// environment includes a valid PATH (e.g., /usr/bin:/usr/local/bin).
			cmd.Env = []string{
				"HOME=" + os.Getenv("HOME"),
				"PATH=" + os.Getenv("PATH"),
				// Prevent git from loading system-wide or user-level config files.
				// An operator-controlled HOME could otherwise redirect git to an
				// attacker-supplied .gitconfig. These variables are supported by
				// git 2.32+ (GIT_CONFIG_COUNT) and older (GIT_CONFIG_NOSYSTEM).
				"GIT_CONFIG_NOSYSTEM=1",
				"GIT_CONFIG_COUNT=0",
			}
			if sshCmd := os.Getenv("GIT_SSH_COMMAND"); sshCmd != "" {
				cmd.Env = append(cmd.Env, "GIT_SSH_COMMAND="+sshCmd)
			} else {
				// Build a default GIT_SSH_COMMAND that pins host fingerprints into a
				// per-server known_hosts file. StrictHostKeyChecking=accept-new records
				// the fingerprint on first connection and rejects changes thereafter,
				// narrowing the DNS rebinding window: an attacker who swaps the DNS
				// record after the initial SSRF check will be blocked by the pinned key.
				// This is a best-effort mitigation — it does not eliminate the window
				// between SSRF validation and the first SSH handshake.
				knownHostsFile := filepath.Join(b.cfg.DataPath, "mirror_known_hosts")
				sshCommand := "ssh -o StrictHostKeyChecking=accept-new -o UserKnownHostsFile=" + knownHostsFile
				if sockPath := os.Getenv("SSH_AUTH_SOCK"); sockPath != "" {
					// SSH_AUTH_SOCK is a Unix socket path from the process environment.
					// It is not influenced by user input (mirror URLs are validated
					// separately), so forwarding it here is safe. A newline in this
					// value would malform the env block; os.Getenv strips NUL bytes
					// but not newlines — reject the value if it contains one.
					if strings.ContainsAny(sockPath, "\n\r\x00") {
						b.logger.Warn("push-mirror: SSH_AUTH_SOCK contains unsafe characters, skipping agent forwarding")
					} else {
						cmd.Env = append(cmd.Env, "SSH_AUTH_SOCK="+sockPath)
					}
				}
				cmd.Env = append(cmd.Env, "GIT_SSH_COMMAND="+sshCommand)
			}
			if out, err := cmd.CombinedOutput(); err != nil {
				b.logger.Warn("push-mirror: push failed", "repo", repo.Name(), "mirror", m.Name, "err", err, "output", string(out))
			} else {
				b.logger.Info("push-mirror: pushed", "repo", repo.Name(), "mirror", m.Name)
			}
		}(m)
	}
	// wg.Wait blocks until all concurrency-bounded pushes complete.
	// PushMirrors itself is called from a goroutine in syncRepoMeta,
	// so blocking here does not delay the push response to the git client.
	wg.Wait()
}

// extractSCPHost parses the host from an SCP-style git remote URL
// (e.g. git@host:repo or host:repo). Returns an error if no host can be
// determined. Strips IPv6 brackets and zone identifiers before returning.
func extractSCPHost(raw string) (string, error) {
	if at := strings.LastIndex(raw, "@"); at != -1 {
		raw = raw[at+1:]
	}
	colon := strings.LastIndex(raw, ":")
	if colon == -1 {
		return "", fmt.Errorf("cannot extract host from SCP-style remote (no colon): %q", raw)
	}
	host := raw[:colon]
	host = strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")
	// Strip IPv6 zone identifier (e.g. "::1%eth0" -> "::1") to prevent
	// scoped addresses from bypassing the loopback check in ValidateHost.
	if z := strings.IndexByte(host, '%'); z != -1 {
		host = host[:z]
	}
	return host, nil
}
