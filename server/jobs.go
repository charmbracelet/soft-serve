package server

import (
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/server/config"
)

var (
	jobSpecs = map[string]string{
		"mirror": "@every 10m",
	}
)

// mirrorJob runs the (pull) mirror job task.
func mirrorJob(cfg *config.Config) func() {
	b := cfg.Backend
	logger := logger.WithPrefix("server.mirrorJob")
	return func() {
		repos, err := b.Repositories()
		if err != nil {
			logger.Error("error getting repositories", "err", err)
			return
		}

		for _, repo := range repos {
			if repo.IsMirror() {
				logger.Info("updating mirror", "repo", repo.Name())
				r, err := repo.Open()
				if err != nil {
					logger.Error("error opening repository", "repo", repo.Name(), "err", err)
					continue
				}

				cmd := git.NewCommand("remote", "update", "--prune")
				cmd.AddEnvs(
					fmt.Sprintf(`GIT_SSH_COMMAND=ssh -o UserKnownHostsFile="%s" -o StrictHostKeyChecking=no -i "%s"`,
						filepath.Join(cfg.DataPath, "ssh", "known_hosts"),
						filepath.Join(cfg.DataPath, cfg.SSH.ClientKeyPath),
					),
				)
				if _, err := cmd.RunInDir(r.Path); err != nil {
					logger.Error("error running git remote update", "repo", repo.Name(), "err", err)
				}
			}
		}
	}
}
