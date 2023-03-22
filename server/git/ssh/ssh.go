package ssh

import (
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/git"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
)

// Middleware adds Git server functionality to the ssh.Server. Repos are stored
// in the specified repo directory. The provided Hooks implementation will be
// checked for access on a per repo basis for a ssh.Session public key.
// Hooks.Push and Hooks.Fetch will be called on successful completion of
// their commands.
func Middleware(cfg *config.Config) wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			func() {
				cmd := s.Command()
				if len(cmd) >= 2 && strings.HasPrefix(cmd[0], "git") {
					gc := cmd[0]
					// repo should be in the form of "repo.git"
					repo := sanitizeRepoName(cmd[1])
					name := repo
					if strings.Contains(repo, "/") {
						log.Printf("invalid repo: %s", repo)
						Fatal(s, fmt.Errorf("%s: %s", git.ErrInvalidRepo, "user repos not supported"))
						return
					}
					pk := s.PublicKey()
					access := cfg.Access.AccessLevel(name, pk)
					// git bare repositories should end in ".git"
					// https://git-scm.com/docs/gitrepository-layout
					repo = strings.TrimSuffix(repo, ".git") + ".git"
					// FIXME: determine repositories path
					repoDir := filepath.Join(cfg.DataPath, "repos", repo)
					switch gc {
					case "git-receive-pack":
						if access < backend.ReadWriteAccess {
							Fatal(s, git.ErrNotAuthed)
							return
						}
						if _, err := cfg.Backend.Repository(name); err != nil {
							if _, err := cfg.Backend.CreateRepository(name, false); err != nil {
								log.Printf("failed to create repo: %s", err)
								Fatal(s, err)
								return
							}
						}
						if err := git.ReceivePack(s, s, s.Stderr(), repoDir); err != nil {
							Fatal(s, git.ErrSystemMalfunction)
						}
						return
					case "git-upload-pack", "git-upload-archive":
						if access < backend.ReadOnlyAccess {
							Fatal(s, git.ErrNotAuthed)
							return
						}
						gitPack := git.UploadPack
						if gc == "git-upload-archive" {
							gitPack = git.UploadArchive
						}
						err := gitPack(s, s, s.Stderr(), repoDir)
						if errors.Is(err, git.ErrInvalidRepo) {
							Fatal(s, git.ErrInvalidRepo)
						} else if err != nil {
							Fatal(s, git.ErrSystemMalfunction)
						}
					}
				}
			}()
			sh(s)
		}
	}
}

// Fatal prints to the session's STDOUT as a git response and exit 1.
func Fatal(s ssh.Session, v ...interface{}) {
	git.WritePktline(s, v...)
	s.Exit(1) // nolint: errcheck
}

func sanitizeRepoName(repo string) string {
	repo = strings.TrimPrefix(repo, "/")
	repo = filepath.Clean(repo)
	repo = strings.TrimSuffix(repo, ".git")
	return repo
}
