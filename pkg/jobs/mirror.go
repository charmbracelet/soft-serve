package jobs

import (
	"bytes"
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
	"github.com/charmbracelet/soft-serve/pkg/store"
	"github.com/charmbracelet/soft-serve/pkg/sync"
	"github.com/spf13/pflag"
)

func init() {
	Register("mirror-pull", mirrorPull{})
}

type (
	mirrorPull       struct{}
	mirrorPullConfig struct {
		baseRunnerConfig
		RepoConfig map[string]string
	}
)

// Description return the description of (pull) mirror job task and implements Runner.
func (m mirrorPull) Description() string {
	return "fetch upstream for mirror repositories"
}

// Config returns the (pull) mirror cronjob configuration and implements Runner.
func (m mirrorPull) Config(ctx context.Context) (RunnerConfig, error) {
	cfg := mirrorPullConfig{
		baseRunnerConfig: baseRunnerConfig{CronSpec: "@every 10m"},
		RepoConfig:       make(map[string]string),
	}
	if spec := config.FromContext(ctx).Jobs.MirrorPull; spec != "" {
		cfg.CronSpec = spec
	}

	return &cfg, nil
}

// Func runs the (pull) mirror job task and implements Runner.
func (m mirrorPull) Func(ctx context.Context, cronCfg RunnerConfig) func() {
	cfg := config.FromContext(ctx)
	logger := log.FromContext(ctx).WithPrefix("jobs.mirror")
	b := backend.FromContext(ctx)
	dbx := db.FromContext(ctx)
	datastore := store.FromContext(ctx)
	jobcfg := cronCfg.(*mirrorPullConfig)
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

					// buffer and write to stdout/stderr in one go,
					// avoiding output confusion through parallel writing.
					var (
						stdout = bytes.NewBuffer(nil)
						stderr = bytes.NewBuffer(nil)
					)
					defer func() {
						jobcfg.Output().Write(stdout.Bytes())
						jobcfg.Error().Write(stderr.Bytes())
					}()

					cmds := []string{
						"fetch --prune",         // fetch prune before updating remote
						"remote update --prune", // update remote and prune remote refs
					}

					gitFlags := []string{}
					for key, val := range jobcfg.RepoConfig {
						gitFlags = append(gitFlags, "-c", key+"="+val)
					}

					for _, c := range cmds {
						args := strings.Split(c, " ")
						args = append(gitFlags, args...)

						cmd := git.NewCommand(args...).WithContext(ctx)
						cmd.AddEnvs(
							fmt.Sprintf(`GIT_SSH_COMMAND=ssh -o UserKnownHostsFile="%s" -o StrictHostKeyChecking=no -i "%s"`,
								filepath.Join(cfg.DataPath, "ssh", "known_hosts"),
								cfg.SSH.ClientKeyPath,
							),
						)

						if _, err := cmd.RunInDir(r.Path); err != nil {
							fmt.Fprintf(stderr, "[%s]: error running git remote update: %v\n", name, err)
							logger.Error("error running git remote update", "repo", name, "err", err)
						}
					}

					if cfg.LFS.Enabled {
						rcfg, err := r.Config()
						if err != nil {
							logger.Error("error getting git config", "repo", name, "err", err)
							fmt.Fprintf(stderr, "[%s]: lfs pull: error getting git config: %v", name, err)
							return
						}

						lfsEndpoint := rcfg.Section("lfs").Option("url")
						if lfsEndpoint == "" {
							// If there is no LFS url defined, means the repo
							// doesn't use LFS and we can skip it.
							return
						}

						ep, err := lfs.NewEndpoint(lfsEndpoint)
						if err != nil {
							logger.Error("error creating LFS endpoint", "repo", name, "err", err)
							fmt.Fprintf(stderr, "[%s]: lfs pull: creating LFS endpoint: %v", name, err)
							return
						}

						client := lfs.NewClient(ep)
						if client == nil {
							fmt.Fprintf(stderr,
								"[%s]: lfs pull: failed to create lfs client: unsupported endpoint %s",
								name, lfsEndpoint)
							logger.Errorf("failed to create lfs client: unsupported endpoint %s", lfsEndpoint)
							return
						}

						if err := backend.StoreRepoMissingLFSObjects(ctx, repo, dbx, datastore, client); err != nil {
							fmt.Fprintf(stderr, "[%s]: lfs pull: failed to store missing lfs objects: %v", name, err)
							logger.Error("failed to store missing lfs objects", "err", err, "path", r.Path)
							return
						}
					}
					fmt.Fprintf(stdout, "[%s]: mirror pull succeed\n", name)
				})
			}
		}

		wq.Run()
	}
}

// FlagSet returns the flag set that can modify configuration values and implements RunnerConfig
func (cfg *mirrorPullConfig) FlagSet() *pflag.FlagSet {
	flags := pflag.NewFlagSet("mirror-pull", pflag.ContinueOnError)
	flags.StringToStringVarP(&cfg.RepoConfig, "config", "c", cfg.RepoConfig, "Override values from git repository configuration files")

	return flags
}
