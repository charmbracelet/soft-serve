package server

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/internal/sync"
)

var (
	jobSpecs = map[string]string{
		"mirror": "@every 10m",
	}
)

// mirrorJob runs the (pull) mirror job task.
func (s *Server) mirrorJob() func() {
	cfg := s.Config
	b := cfg.Backend
	logger := s.logger
	return func() {
		repos, err := b.Repositories()
		if err != nil {
			logger.Error("error getting repositories", "err", err)
			return
		}

		// Divide the work up among the number of CPUs.
		wq := sync.NewWorkQueue(runtime.NumCPU() / 4)

		for _, repo := range repos {
			if repo.IsMirror() {
				logger.Debug("updating mirror", "repo", repo.Name())
				r, err := repo.Open()
				if err != nil {
					logger.Error("error opening repository", "repo", repo.Name(), "err", err)
					return
				}

				name := repo.Name()
				wq.Add(name, func() {
					cmd := git.NewCommand("remote", "update", "--prune")
					cmd.AddEnvs(
						fmt.Sprintf(`GIT_SSH_COMMAND=ssh -o UserKnownHostsFile="%s" -o StrictHostKeyChecking=no -i "%s"`,
							filepath.Join(cfg.DataPath, "ssh", "known_hosts"),
							cfg.SSH.ClientKeyPath,
						),
					)
					if _, err := cmd.RunInDir(r.Path); err != nil {
						logger.Error("error running git remote update", "repo", name, "err", err)
					}

					logger.Debug("mirror updated", "repo", name)
				})
			}
		}

		wq.Run()
	}
}
