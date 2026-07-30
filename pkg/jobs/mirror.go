package jobs

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"charm.land/log/v2"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/lfs"
	"github.com/charmbracelet/soft-serve/pkg/ssrf"
	"github.com/charmbracelet/soft-serve/pkg/store"
	"github.com/charmbracelet/soft-serve/pkg/sync"
)

func init() {
	Register("mirror-pull", mirrorPull{})
}

type mirrorPull struct{}

// validateMirrorRemotes validates every remote configured on a mirror
// repository and returns the git environment that must be applied to the sync
// commands.
//
// `git remote update` fetches from all remotes, not just origin, so every one
// of them has to pass. Any failure to read or validate skips the sync: a
// remote that cannot be checked is not one to fetch from.
func validateMirrorRemotes(r *git.Repository) ([]string, error) {
	cfg, err := r.Config()
	if err != nil {
		return nil, fmt.Errorf("reading git config: %w", err)
	}

	var remotes []ssrf.ValidatedGitRemote
	for _, sub := range cfg.Section("remote").Subsections {
		url := sub.Option("url")
		if url == "" {
			continue
		}

		v, err := ssrf.ValidateGitRemote(url)
		if err != nil {
			return nil, fmt.Errorf("remote %q: %w", sub.Name, err)
		}
		remotes = append(remotes, v)
	}

	if len(remotes) == 0 {
		return nil, fmt.Errorf("no remote url configured")
	}

	return ssrf.GitEnv(remotes...), nil
}

// Spec derives the spec used for pull mirrors and implements Runner.
func (m mirrorPull) Spec(ctx context.Context) string {
	cfg := config.FromContext(ctx)
	if cfg.Jobs.MirrorPull != "" {
		return cfg.Jobs.MirrorPull
	}
	return "@every 10m"
}

// Func runs the (pull) mirror job task and implements Runner.
func (m mirrorPull) Func(ctx context.Context) func() {
	cfg := config.FromContext(ctx)
	logger := log.FromContext(ctx).WithPrefix("jobs.mirror")
	b := backend.FromContext(ctx)
	dbx := db.FromContext(ctx)
	datastore := store.FromContext(ctx)
	return func() {
		repos, err := b.Repositories(ctx)
		if err != nil {
			logger.Error("error getting repositories", "err", err)
			return
		}

		// Divide the work up among the number of CPUs.
		wq := sync.NewWorkPool(ctx, runtime.GOMAXPROCS(0),
			sync.WithWorkPoolLogger(logger.Errorf),
		)

		logger.Debug("updating mirror repos")
		for _, repo := range repos {
			if repo.IsMirror() {
				r, err := repo.Open()
				if err != nil {
					logger.Error("error opening repository", "repo", repo.Name(), "err", err)
					continue
				}

				name := repo.Name()
				wq.Add(name, func() {
					repo := repo

					// Re-validate every configured remote before syncing.
					// `remote update` touches all of them, and a remote may
					// predate the import-time guard or have been written out
					// of band. Validation failure skips the repo entirely.
					remoteEnv, err := validateMirrorRemotes(r)
					if err != nil {
						logger.Warn("skipping mirror sync, remote failed validation", "repo", name, "err", err)
						return
					}

					// remoteEnv carries the SSRF guard and must reach every
					// command below, so build the full set once.
					syncEnv := append(remoteEnv,
						fmt.Sprintf(`GIT_SSH_COMMAND=ssh -o UserKnownHostsFile="%s" -o StrictHostKeyChecking=no -i "%s"`,
							filepath.Join(cfg.DataPath, "ssh", "known_hosts"),
							cfg.SSH.ClientKeyPath,
						),
					)

					cmds := []string{
						"fetch --prune",         // fetch prune before updating remote
						"remote update --prune", // update remote and prune remote refs
					}

					for _, c := range cmds {
						args := strings.Split(c, " ")
						cmd := git.NewCommand(args...).WithContext(ctx).WithTimeout(-1)
						cmd.AddEnvs(syncEnv...)

						if _, err := cmd.RunInDir(r.Path); err != nil {
							logger.Error("error running git remote update", "repo", name, "err", err)
						}
					}

					if cfg.LFS.Enabled {
						rcfg, err := r.Config()
						if err != nil {
							logger.Error("error getting git config", "repo", name, "err", err)
							return
						}

						lfsEndpoint := rcfg.Section("lfs").Option("url")
						if lfsEndpoint == "" {
							// If there is no LFS url defined, means the repo
							// doesn't use LFS and we can skip it.
							return
						}

						// The endpoint is stored in the repo config and may
						// predate validation, so check it before dialing.
						if _, err := ssrf.ValidateGitRemote(lfsEndpoint); err != nil {
							logger.Warn("skipping lfs sync, endpoint failed validation", "repo", name, "err", err)
							return
						}

						ep, err := lfs.NewEndpoint(lfsEndpoint)
						if err != nil {
							logger.Error("error creating LFS endpoint", "repo", name, "err", err)
							return
						}

						client := lfs.NewClient(ep)
						if client == nil {
							logger.Errorf("failed to create lfs client: unsupported endpoint %s", lfsEndpoint)
							return
						}

						if err := backend.StoreRepoMissingLFSObjects(ctx, repo, dbx, datastore, client); err != nil {
							logger.Error("failed to store missing lfs objects", "err", err, "path", r.Path)
							return
						}
					}
				})
			}
		}

		wq.Run()
	}
}
