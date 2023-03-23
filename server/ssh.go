package server

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/server/backend"
	cm "github.com/charmbracelet/soft-serve/server/cmd"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	lm "github.com/charmbracelet/wish/logging"
	rm "github.com/charmbracelet/wish/recover"
	"github.com/muesli/termenv"
	gossh "golang.org/x/crypto/ssh"
)

// SSHServer is a SSH server that implements the git protocol.
type SSHServer struct {
	*ssh.Server
	cfg *config.Config
}

// NewSSHServer returns a new SSHServer.
func NewSSHServer(cfg *config.Config) (*SSHServer, error) {
	var err error
	s := &SSHServer{cfg: cfg}
	logger := logger.StandardLog(log.StandardLogOptions{ForceLevel: log.DebugLevel})
	mw := []wish.Middleware{
		rm.MiddlewareWithLogger(
			logger,
			// BubbleTea middleware.
			bm.MiddlewareWithProgramHandler(SessionHandler(cfg), termenv.ANSI256),
			// Command middleware must come after the git middleware.
			cm.Middleware(cfg),
			// Git middleware.
			s.Middleware(cfg),
			// Logging middleware.
			lm.MiddlewareWithLogger(logger),
		),
	}
	s.Server, err = wish.NewServer(
		ssh.PublicKeyAuth(s.PublicKeyHandler),
		ssh.KeyboardInteractiveAuth(s.KeyboardInteractiveHandler),
		wish.WithAddress(cfg.SSH.ListenAddr),
		wish.WithHostKeyPath(cfg.SSH.KeyPath),
		wish.WithMiddleware(mw...),
	)
	if err != nil {
		return nil, err
	}

	if cfg.SSH.MaxTimeout > 0 {
		s.Server.MaxTimeout = time.Duration(cfg.SSH.MaxTimeout) * time.Second
	}
	if cfg.SSH.IdleTimeout > 0 {
		s.Server.IdleTimeout = time.Duration(cfg.SSH.IdleTimeout) * time.Second
	}

	return s, nil
}

// PublicKeyAuthHandler handles public key authentication.
func (s *SSHServer) PublicKeyHandler(ctx ssh.Context, pk ssh.PublicKey) bool {
	return s.cfg.Access.AccessLevel("", pk) > backend.NoAccess
}

// KeyboardInteractiveHandler handles keyboard interactive authentication.
func (s *SSHServer) KeyboardInteractiveHandler(_ ssh.Context, _ gossh.KeyboardInteractiveChallenge) bool {
	return true
}

// Middleware adds Git server functionality to the ssh.Server. Repos are stored
// in the specified repo directory. The provided Hooks implementation will be
// checked for access on a per repo basis for a ssh.Session public key.
// Hooks.Push and Hooks.Fetch will be called on successful completion of
// their commands.
func (s *SSHServer) Middleware(cfg *config.Config) wish.Middleware {
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
						sshFatal(s, fmt.Errorf("%s: %s", ErrInvalidRepo, "user repos not supported"))
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
					case ReceivePackBin:
						if access < backend.ReadWriteAccess {
							sshFatal(s, ErrNotAuthed)
							return
						}
						if _, err := cfg.Backend.Repository(name); err != nil {
							if _, err := cfg.Backend.CreateRepository(name, false); err != nil {
								log.Printf("failed to create repo: %s", err)
								sshFatal(s, err)
								return
							}
						}
						if err := ReceivePack(s, s, s.Stderr(), repoDir); err != nil {
							sshFatal(s, ErrSystemMalfunction)
						}
						return
					case UploadPackBin, UploadArchiveBin:
						if access < backend.ReadOnlyAccess {
							sshFatal(s, ErrNotAuthed)
							return
						}
						gitPack := UploadPack
						if gc == UploadArchiveBin {
							gitPack = UploadArchive
						}
						err := gitPack(s, s, s.Stderr(), repoDir)
						if errors.Is(err, ErrInvalidRepo) {
							sshFatal(s, ErrInvalidRepo)
						} else if err != nil {
							sshFatal(s, ErrSystemMalfunction)
						}
					}
				}
			}()
			sh(s)
		}
	}
}

// sshFatal prints to the session's STDOUT as a git response and exit 1.
func sshFatal(s ssh.Session, v ...interface{}) {
	WritePktline(s, v...)
	s.Exit(1) // nolint: errcheck
}

func sanitizeRepoName(repo string) string {
	repo = strings.TrimPrefix(repo, "/")
	repo = filepath.Clean(repo)
	repo = strings.TrimSuffix(repo, ".git")
	return repo
}
