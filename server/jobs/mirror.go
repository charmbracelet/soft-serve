package jobs

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/sync"
)

func init() {
	Register("mirror-pull", "@every 10m", mirrorPull)
}

// mirrorPull runs the (pull) mirror job task.
func mirrorPull(ctx context.Context) func() {
	cfg := config.FromContext(ctx)
	logger := log.FromContext(ctx).WithPrefix("jobs.mirror")
	b := backend.FromContext(ctx)
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
					cmd := git.NewCommand("remote", "update", "--prune").WithContext(ctx)
					cmd.AddEnvs(
						fmt.Sprintf(`GIT_SSH_COMMAND=ssh -o UserKnownHostsFile="%s" -o StrictHostKeyChecking=no -i "%s"`,
							filepath.Join(cfg.DataPath, "ssh", "known_hosts"),
							cfg.SSH.ClientKeyPath,
						),
					)

					if _, err := cmd.RunInDir(r.Path); err != nil {
						logger.Error("error running git remote update", "repo", name, "err", err)
					}
				})
			}
		}

		wq.Run()
	}
}
