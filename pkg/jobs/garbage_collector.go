package jobs

import (
	"context"
	"runtime"

	"charm.land/log/v2"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/sync"
)

func init() {
	Register("git-gc", gitGC{})
}

type gitGC struct{}

func (g gitGC) Spec(ctx context.Context) string {
	cfg := config.FromContext(ctx)
	if cfg.Jobs.GitGC != "" {
		return cfg.Jobs.GitGC
	}
	return ""
}

func (g gitGC) Func(ctx context.Context) func() {
	b := backend.FromContext(ctx)
	logger := log.FromContext(ctx).WithPrefix("jobs.gitgc")
	return func() {
		repos, err := b.Repositories(ctx)
		if err != nil {
			logger.Error("error getting repositories", "err", err)
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
				continue
			}

			name := repo.Name()
			wq.Add(name, func() {
				cmd := git.NewCommand("gc").WithContext(ctx)
				if _, err := cmd.RunInDir(r.Path); err != nil {
					logger.Error("error running git remote update", "repo", name, "err", err)
				}

			})

			// TODO: clean up lfs
		}

		wq.Run()
	}
}
