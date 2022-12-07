package ssh

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/soft-serve/proto"
	"github.com/charmbracelet/soft-serve/server/git"
	"github.com/charmbracelet/wish"
	"github.com/gliderlabs/ssh"
)

// Auth is the interface that wraps both Access and Provider interfaces.
type Auth interface {
	proto.Access
	proto.Provider
}

// Middleware adds Git server functionality to the ssh.Server. Repos are stored
// in the specified repo directory. The provided Hooks implementation will be
// checked for access on a per repo basis for a ssh.Session public key.
// Hooks.Push and Hooks.Fetch will be called on successful completion of
// their commands.
func Middleware(repoDir string, auth Auth) wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			func() {
				cmd := s.Command()
				if len(cmd) == 2 && strings.HasPrefix(cmd[0], "git") {
					gc := cmd[0]
					// repo should be in the form of "repo.git"
					repo := strings.TrimPrefix(cmd[1], "/")
					repo = filepath.Clean(repo)
					name := repo
					if strings.Contains(repo, "/") {
						log.Printf("invalid repo: %s", repo)
						Fatal(s, fmt.Errorf("%s: %s", git.ErrInvalidRepo, "user repos not supported"))
						return
					}
					pk := s.PublicKey()
					access := auth.AuthRepo(name, pk)
					// git bare repositories should end in ".git"
					// https://git-scm.com/docs/gitrepository-layout
					repo = strings.TrimSuffix(repo, ".git") + ".git"
					switch gc {
					case "git-receive-pack":
						switch access {
						case proto.ReadWriteAccess, proto.AdminAccess:
							if _, err := auth.Open(name); err != nil {
								if err := auth.Create(name, "", "", false); err != nil {
									log.Printf("failed to create repo: %s", err)
									Fatal(s, err)
									return
								}
							}
							if err := git.GitPack(s, s, s.Stderr(), gc, repoDir, repo); err != nil {
								Fatal(s, git.ErrSystemMalfunction)
							}
						default:
							Fatal(s, git.ErrNotAuthed)
						}
						return
					case "git-upload-archive", "git-upload-pack":
						log.Printf("access %s", access)
						switch access {
						case proto.ReadOnlyAccess, proto.ReadWriteAccess, proto.AdminAccess:
							// try to upload <repo>.git first, then <repo>
							err := git.GitPack(s, s, s.Stderr(), gc, repoDir, repo)
							if err != nil {
								err = git.GitPack(s, s, s.Stderr(), gc, repoDir, strings.TrimSuffix(repo, ".git"))
							}
							switch err {
							case git.ErrInvalidRepo:
								Fatal(s, git.ErrInvalidRepo)
							case nil:
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
