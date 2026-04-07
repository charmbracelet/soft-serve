package backend

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/pkg/hooks"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/sshutils"
	"github.com/charmbracelet/soft-serve/pkg/webhook"
	"gopkg.in/yaml.v3"
)

var _ hooks.Hooks = (*Backend)(nil)

// repoMetaConfig holds repository metadata fields synced from .soft-serve.yaml.
type repoMetaConfig struct {
	Description string `yaml:"description"`
	Private     *bool  `yaml:"private"`
	Hidden      *bool  `yaml:"hidden"`
}

// PostReceive is called by the git post-receive hook. It implements Hooks.
// Metadata sync (.soft-serve.yaml) is performed asynchronously so the push
// response is not blocked by DB writes or git tree reads.
func (d *Backend) PostReceive(ctx context.Context, _ io.Writer, _ io.Writer, repo string, args []hooks.HookArg) {
	d.logger.Debug("post-receive hook called", "repo", repo, "args", args)

	// Capture the pushing user before the goroutine is launched so that the
	// caller's context (which may be cancelled after the push completes) does
	// not need to remain valid inside the goroutine.
	user := proto.UserFromContext(ctx)

	// Sync .soft-serve.yaml metadata asynchronously so the push
	// response is not blocked by DB writes or git tree reads.
	go func() {
		// The 30-second timeout covers metadata sync (DB writes, YAML parse).
		// PushMirrors is called from syncRepoMeta with d.ctx (the backend root
		// context, not syncCtx), so mirror pushes are NOT constrained by this
		// timeout — they run with their own per-push mirrorPushTimeout.
		syncCtx, cancel := context.WithTimeout(d.ctx, 30*time.Second)
		defer cancel()
		d.syncRepoMeta(syncCtx, repo, user)
	}()
}

// syncRepoMeta reads .soft-serve.yaml from HEAD and applies non-zero fields
// to the repository backend. Private and hidden require admin access.
// user is the pushing user captured from the caller's context before the
// goroutine was launched.
func (d *Backend) syncRepoMeta(ctx context.Context, repo string, user proto.User) {
	r, err := d.Repository(ctx, repo)
	if err != nil {
		d.logger.Warn("post-receive: failed to find repository", "repo", repo, "err", err)
		return
	}

	// Run push mirrors with a separate context derived from the backend's root
	// context so they are not limited by the 30-second syncCtx. Each mirror
	// goroutine inside PushMirrors creates its own per-push timeout via
	// mirrorPushTimeout; using the backend root context here lets that inner
	// timeout operate at full duration.
	d.PushMirrors(d.ctx, r)

	gr, err := r.Open()
	if err != nil {
		// empty or invalid repo — skip
		return
	}

	const maxMetaFileSize = 64 * 1024 // 64 KB

	content, _, err := git.LatestFile(gr, nil, ".soft-serve.yaml")
	if err != nil {
		// file absent or no commits yet — no-op
		return
	}
	if len(content) > maxMetaFileSize {
		d.logger.Warnf("post-receive: .soft-serve.yaml exceeds %d bytes, skipping", maxMetaFileSize)
		return
	}

	var meta repoMetaConfig
	if err := yaml.Unmarshal([]byte(content), &meta); err != nil {
		d.logger.Warnf("post-receive: parse .soft-serve.yaml: %v", err)
		return
	}

	// Only admins may change visibility. Use the user captured from the
	// caller's context rather than looking it up from d.ctx, which would
	// return nil because d.ctx has no user attached.
	var isAdmin bool
	if user != nil {
		isAdmin = user.IsAdmin()
	}

	if meta.Description != "" {
		const maxDescLen = 2048
		desc := meta.Description
		if runes := []rune(desc); len(runes) > maxDescLen {
			desc = string(runes[:maxDescLen])
		}
		if err := d.SetDescription(ctx, repo, desc); err != nil {
			d.logger.Warnf("post-receive: set description: %v", err)
		}
	}

	if isAdmin {
		if meta.Private != nil {
			if err := d.SetPrivate(ctx, repo, *meta.Private); err != nil {
				d.logger.Warnf("post-receive: set private: %v", err)
			}
		}
		if meta.Hidden != nil {
			if err := d.SetHidden(ctx, repo, *meta.Hidden); err != nil {
				d.logger.Warnf("post-receive: set hidden: %v", err)
			}
		}
	}
}

// PreReceive is called by the git pre-receive hook.
//
// It implements Hooks.
func (d *Backend) PreReceive(_ context.Context, _ io.Writer, _ io.Writer, repo string, args []hooks.HookArg) {
	d.logger.Debug("pre-receive hook called", "repo", repo, "args", args)
}

// Update is called by the git update hook.
//
// It implements Hooks.
func (d *Backend) Update(ctx context.Context, _ io.Writer, _ io.Writer, repo string, arg hooks.HookArg) {
	d.logger.Debug("update hook called", "repo", repo, "arg", arg)

	// Find user from hook environment variables. These are process-global but
	// safe because each hook invocation is run in a separate subprocess (see
	// pkg/hooks/gen.go); concurrent pushes in the same server process never
	// share the hook subprocess's environment.
	var user proto.User
	if pubkey := os.Getenv("SOFT_SERVE_PUBLIC_KEY"); pubkey != "" {
		pk, _, err := sshutils.ParseAuthorizedKey(pubkey)
		if err != nil {
			d.logger.Error("error parsing public key", "err", err)
			return
		}

		user, err = d.UserByPublicKey(ctx, pk)
		if err != nil {
			d.logger.Error("error finding user from public key", "key", pubkey, "err", err)
			return
		}
	} else if username := os.Getenv("SOFT_SERVE_USERNAME"); username != "" {
		var err error
		user, err = d.User(ctx, username)
		if err != nil {
			d.logger.Error("error finding user from username", "username", username, "err", err)
			return
		}
	} else {
		d.logger.Warn("error finding user: neither SOFT_SERVE_PUBLIC_KEY nor SOFT_SERVE_USERNAME is set in hook environment")
		return
	}

	// Get repo
	r, err := d.Repository(ctx, repo)
	if err != nil {
		d.logger.Error("error finding repository", "repo", repo, "err", err)
		return
	}

	// Webhook delivery runs synchronously in the update hook subprocess.
	// Async dispatch would require an IPC channel between the hook process
	// and the main server; not currently implemented.
	if git.IsZeroHash(arg.OldSha) || git.IsZeroHash(arg.NewSha) {
		wh, err := webhook.NewBranchTagEvent(ctx, user, r, arg.RefName, arg.OldSha, arg.NewSha)
		if err != nil {
			d.logger.Error("error creating branch_tag webhook", "err", err)
		} else if err := webhook.SendEvent(ctx, wh); err != nil {
			d.logger.Error("error sending branch_tag webhook", "err", err)
		}
	}
	wh, err := webhook.NewPushEvent(ctx, user, r, arg.RefName, arg.OldSha, arg.NewSha)
	if err != nil {
		d.logger.Error("error creating push webhook", "err", err)
	} else if err := webhook.SendEvent(ctx, wh); err != nil {
		d.logger.Error("error sending push webhook", "err", err)
	}
}

// PostUpdate is called by the git post-update hook.
//
// It implements Hooks.
func (d *Backend) PostUpdate(ctx context.Context, _ io.Writer, _ io.Writer, repo string, args ...string) {
	d.logger.Debug("post-update hook called", "repo", repo, "args", args)

	if err := populateLastModified(ctx, d, repo); err != nil {
		d.logger.Error("error populating last-modified", "repo", repo, "err", err)
	}
}

func populateLastModified(ctx context.Context, d *Backend, name string) error {
	var rr *repo
	_rr, err := d.Repository(ctx, name)
	if err != nil {
		return err
	}

	if r, ok := _rr.(*repo); ok {
		rr = r
	} else {
		return proto.ErrRepoNotFound
	}

	r, err := rr.Open()
	if err != nil {
		return err
	}

	c, err := r.LatestCommitTime()
	if err != nil {
		return err
	}

	return rr.writeLastModified(c)
}
