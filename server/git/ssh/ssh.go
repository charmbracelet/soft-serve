package ssh

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/soft-serve/server/git"
	"github.com/charmbracelet/wish"
	"github.com/gliderlabs/ssh"
)

// Middleware adds Git server functionality to the ssh.Server. Repos are stored
// in the specified repo directory. The provided Hooks implementation will be
// checked for access on a per repo basis for a ssh.Session public key.
// Hooks.Push and Hooks.Fetch will be called on successful completion of
// their commands.
func Middleware(repoDir string, gh git.Hooks) wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			func() {
				cmd := s.Command()
				if len(cmd) == 2 && strings.HasPrefix(cmd[0], "git") {
					gc := cmd[0]
					// repo should be in the form of "repo.git"
					repo := strings.TrimPrefix(cmd[1], "/")
					repo = filepath.Clean(repo)
					if strings.Contains(repo, "/") {
						log.Printf("invalid repo: %s", repo)
						Fatal(s, fmt.Errorf("%s: %s", git.ErrInvalidRepo, "user repos not supported"))
						return
					}
					pk := s.PublicKey()
					access := gh.AuthRepo(strings.TrimSuffix(repo, ".git"), pk)
					// git bare repositories should end in ".git"
					// https://git-scm.com/docs/gitrepository-layout
					if !strings.HasSuffix(repo, ".git") {
						repo += ".git"
					}
					switch gc {
					case "git-receive-pack":
						switch access {
						case git.ReadWriteAccess, git.AdminAccess:
							err := git.GitPack(s, s, s.Stderr(), gc, repoDir, repo)
							if err != nil {
								Fatal(s, git.ErrSystemMalfunction)
							} else {
								gh.Push(repo, pk)
							}
						default:
							Fatal(s, git.ErrNotAuthed)
						}
						return
					case "git-upload-archive", "git-upload-pack":
						switch access {
						case git.ReadOnlyAccess, git.ReadWriteAccess, git.AdminAccess:
							// try to upload <repo>.git first, then <repo>
							err := git.GitPack(s, s, s.Stderr(), gc, repoDir, repo)
							if err != nil {
								err = git.GitPack(s, s, s.Stderr(), gc, repoDir, strings.TrimSuffix(repo, ".git"))
							}
							switch err {
							case git.ErrInvalidRepo:
								Fatal(s, git.ErrInvalidRepo)
							case nil:
								gh.Fetch(repo, pk)
							default:
								log.Printf("unknown git error: %s", err)
								Fatal(s, git.ErrSystemMalfunction)
							}
						default:
							Fatal(s, git.ErrNotAuthed)
						}
						return
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
