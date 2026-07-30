package jobs

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"charm.land/log/v2"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/lfs"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/storage"
	"github.com/charmbracelet/soft-serve/pkg/store"
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

		RepoConfig     map[string]string
		LFSPruneExpire time.Duration
		Aggressive     bool
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
		LFSPruneExpire:   time.Hour * 24 * 7,
	}

	if spec := config.FromContext(ctx).Jobs.GitGC; spec != "" {
		cfg.CronSpec = spec
	}

	return &cfg, nil
}

// cleanup return garbage collection work function on a repository added to sync.WorkPool
func (g gitGC) cleanup(ctx context.Context, jobcfg *gitGCConfig, repo proto.Repository) func() {
	repoName := repo.Name()

	b := backend.FromContext(ctx)
	datastore := store.FromContext(ctx)
	dbx := db.FromContext(ctx)
	cfg := config.FromContext(ctx)

	logger := log.FromContext(ctx).WithPrefix("jobs.gitgc")
	logger = logger.With("repo", repoName)

	return func() {
		r, err := repo.Open()
		if err != nil {
			logger.Error("error opening repository", "err", err)
			fmt.Fprintf(jobcfg.Error(), "[%s] error opening repository: %v\n", repoName, err)
			return
		}

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

		logger.Debug("start git garbage collection")

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
			logger.Error("error running git remote update", "err", err)
			fmt.Fprintf(stderr, "[%s] git gc failed: %v\n", repoName, err)
			return
		}

		// clean up unreachable lfs objects
		if cfg.LFS.Enabled {
			logger.Debug("start lfs objects garbage collection")
			pruneBefore := time.Now().Add(-jobcfg.LFSPruneExpire)
			repoID := strconv.FormatInt(repo.ID(), 10)
			strg := storage.NewLocalStorage(filepath.Join(cfg.DataPath, "lfs", repoID))

			objs, err := b.GetUnreachableLFSObjects(ctx, repoName)
			if err != nil {
				logger.Error("error get unreachable lfs objects", "err", err)
				fmt.Fprintf(stderr, "[%s] get unreachable lfs objects: %v\n", repoName, err)
				return
			}

			for _, obj := range objs {
				if obj.UpdatedAt.Before(pruneBefore) {
					if err := dbx.TransactionContext(ctx, func(tx *db.Tx) error {
						if err := datastore.DeleteLFSObjectByOid(ctx, dbx, repo.ID(), obj.Oid); err != nil {
							return err
						}
						p := lfs.Pointer{Oid: obj.Oid}
						return strg.Delete(filepath.Join("objects", p.RelativePath()))
					}); err != nil {
						logger.Error("error clear lfs objects", "err", err, "oid", obj.Oid)
						fmt.Fprintf(stderr, "[%s] error delete lfs object %s: %v\n", repoName, obj.Oid, err)
					} else {
						logger.Info("removed unreachable lfs object", "repo", repoName, "oid", obj.Oid)
						fmt.Fprintf(stdout, "[%s] removed lfs object %s\n", repoName, obj.Oid)
					}
				}
			}
		}
		fmt.Fprintf(stdout, "[%s] git gc successful\n", repoName)
	}
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

		for _, repo := range repos {
			name := repo.Name()
			wq.Add(name, g.cleanup(ctx, jobcfg, repo))
		}

		wq.Run()
	}
}

// FlagSet returns the flag set that can modify configuration values and implements RunnerConfig
func (cfg *gitGCConfig) FlagSet() *pflag.FlagSet {
	flags := pflag.NewFlagSet("git-gc", pflag.ContinueOnError)
	flags.StringToStringVarP(&cfg.RepoConfig, "config", "c", cfg.RepoConfig, "Override values from git repository configuration files")

	flags.BoolVar(&cfg.Aggressive, "aggressive", cfg.Aggressive, "Optimize the repository more aggressively, see git-gc(1) for more details")
	flags.DurationVar(&cfg.LFSPruneExpire, "lfsprune-expire", cfg.LFSPruneExpire, "Expire duration of unreachable LFS objects following the latest update")

	return flags
}
