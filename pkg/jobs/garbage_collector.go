package jobs

import (
	"bytes"
	"context"
	"fmt"
	"runtime"

	"charm.land/log/v2"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/sync"
	"github.com/spf13/pflag"
)

func init() {
	Register("git-gc", gitGC{})
}

type (
	gitGC       struct{}
	gitGCConfig struct {
		baseRunnerConfig

		RepoConfig map[string]string
		Aggressive bool
	}
)

// Description return the description of garbage collector job task and implements Runner.
func (m gitGC) Description() string {
	return "clean up the garbage in repositories"
}

// Config returns the garbage collector job task configuration and implements Runner.
func (m gitGC) Config(ctx context.Context) (RunnerConfig, error) {
	cfg := gitGCConfig{
		baseRunnerConfig: baseRunnerConfig{CronSpec: ""},
		Aggressive:       false,
		RepoConfig:       make(map[string]string),
	}

	if spec := config.FromContext(ctx).Jobs.GitGC; spec != "" {
		cfg.CronSpec = spec
	}

	return &cfg, nil
}

// Func runs the garbage collector job task and implements Runner.
func (g gitGC) Func(ctx context.Context, cronCfg RunnerConfig) func() {
	b := backend.FromContext(ctx)
	logger := log.FromContext(ctx).WithPrefix("jobs.gitgc")
	jobcfg := cronCfg.(*gitGCConfig)

	return func() {
		repos, err := b.Repositories(ctx)
		if err != nil {
			logger.Error("error getting repositories", "err", err)
			fmt.Fprintf(jobcfg.Error(), "error getting repositories: %v\n", err)
			return
		}

		wq := sync.NewWorkPool(ctx, runtime.GOMAXPROCS(0),
			sync.WithWorkPoolLogger(logger.Errorf),
		)

		logger.Debug("cleaning git garbage")
		for _, repo := range repos {
			r, err := repo.Open()
			if err != nil {
				logger.Error("error opening repository", "repo", repo.Name(), "err", err)
				fmt.Fprintf(jobcfg.Error(), "[%s] error opening repository: %v\n", repo.Name(), err)
				continue
			}

			name := repo.Name()
			wq.Add(name, func() {
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

				var cmdArgs []string = nil
				for key, val := range jobcfg.RepoConfig {
					cmdArgs = append(cmdArgs, "-c", key+"="+val)
				}

				cmdArgs = append(cmdArgs, "gc")

				if jobcfg.Aggressive {
					cmdArgs = append(cmdArgs, "--aggressive")
				}

				// `git gc` would not output anything if no tty
				cmd := git.NewCommand(cmdArgs...).WithContext(ctx)
				if _, err := cmd.RunInDir(r.Path); err != nil {
					logger.Error("error running git remote update", "repo", name, "err", err)
					fmt.Fprintf(stderr, "[%s] git gc failed: %v\n", name, err)
				} else {
					fmt.Fprintf(stdout, "[%s] git gc succeed\n", name)
				}
			})

			// TODO: clean up lfs
		}

		wq.Run()
	}
}

// FlagSet returns the flag set that can modify configuration values and implements RunnerConfig
func (cfg *gitGCConfig) FlagSet() *pflag.FlagSet {
	flags := pflag.NewFlagSet("git-gc", pflag.ContinueOnError)
	flags.StringToStringVarP(&cfg.RepoConfig, "config", "c", cfg.RepoConfig, "Override values from git repository configuration files")

	flags.BoolVar(&cfg.Aggressive, "aggressive", cfg.Aggressive, "Optimize the repository more aggressively, see git-gc(1) for more details")

	return flags
}
