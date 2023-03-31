package server

import (
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/server/backend"
)

var (
	jobSpecs = map[string]string{
		"mirror": "@every 10m",
	}
)

// mirrorJob runs the (pull) mirror job task.
func mirrorJob(b backend.Backend) func() {
	logger := logger.WithPrefix("server.mirrorJob")
	return func() {
		repos, err := b.Repositories()
		if err != nil {
			logger.Error("error getting repositories", "err", err)
			return
		}

		for _, repo := range repos {
			if repo.IsMirror() {
				logger.Debug("updating mirror", "repo", repo.Name())
				r, err := repo.Open()
				if err != nil {
					logger.Error("error opening repository", "repo", repo.Name(), "err", err)
					continue
				}

				cmd := git.NewCommand("remote", "update", "--prune")
				if _, err := cmd.RunInDir(r.Path); err != nil {
					logger.Error("error running git remote update", "repo", repo.Name(), "err", err)
				}
			}
		}
	}
}
