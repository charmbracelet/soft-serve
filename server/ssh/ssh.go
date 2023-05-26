package ssh

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/keygen"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/server/backend"
	cm "github.com/charmbracelet/soft-serve/server/cmd"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/git"
	"github.com/charmbracelet/soft-serve/server/utils"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	lm "github.com/charmbracelet/wish/logging"
	rm "github.com/charmbracelet/wish/recover"
	"github.com/muesli/termenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	gossh "golang.org/x/crypto/ssh"
)

var (
	publicKeyCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "soft_serve",
		Subsystem: "ssh",
		Name:      "public_key_auth_total",
		Help:      "The total number of public key auth requests",
	}, []string{"key", "user", "allowed"})

	noAuthCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "soft_serve",
		Subsystem: "ssh",
		Name:      "none_auth_total",
		Help:      "The total number of none auth requests",
	}, []string{"user", "allowed"})

	keyboardInteractiveCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "soft_serve",
		Subsystem: "ssh",
		Name:      "keyboard_interactive_auth_total",
		Help:      "The total number of keyboard interactive auth requests",
	}, []string{"user", "allowed"})

	uploadPackCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "soft_serve",
		Subsystem: "ssh",
		Name:      "git_upload_pack_total",
		Help:      "The total number of git-upload-pack requests",
	}, []string{"key", "user", "repo"})

	receivePackCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "soft_serve",
		Subsystem: "ssh",
		Name:      "git_receive_pack_total",
		Help:      "The total number of git-receive-pack requests",
	}, []string{"key", "user", "repo"})

	uploadArchiveCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "soft_serve",
		Subsystem: "ssh",
		Name:      "git_upload_archive_total",
		Help:      "The total number of git-upload-archive requests",
	}, []string{"key", "user", "repo"})

	createRepoCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "soft_serve",
		Subsystem: "ssh",
		Name:      "create_repo_total",
		Help:      "The total number of create repo requests",
	}, []string{"key", "user", "repo"})
)

// SSHServer is a SSH server that implements the git protocol.
type SSHServer struct {
	srv    *ssh.Server
	cfg    *config.Config
	be     backend.Backend
	ctx    context.Context
	logger *log.Logger
}

// NewSSHServer returns a new SSHServer.
func NewSSHServer(ctx context.Context) (*SSHServer, error) {
	cfg := config.FromContext(ctx)
	logger := log.FromContext(ctx).WithPrefix("ssh")

	var err error
	s := &SSHServer{
		cfg:    cfg,
		ctx:    ctx,
		be:     backend.FromContext(ctx),
		logger: logger,
	}

	mw := []wish.Middleware{
		rm.MiddlewareWithLogger(
			logger,
			// BubbleTea middleware.
			bm.MiddlewareWithProgramHandler(SessionHandler(cfg), termenv.ANSI256),
			// CLI middleware.
			cm.Middleware(cfg, logger),
			// Git middleware.
			s.Middleware(cfg),
			// Logging middleware.
			lm.MiddlewareWithLogger(logger.
				StandardLog(log.StandardLogOptions{ForceLevel: log.DebugLevel})),
		),
	}

	s.srv, err = wish.NewServer(
		ssh.PublicKeyAuth(s.PublicKeyHandler),
		ssh.KeyboardInteractiveAuth(s.KeyboardInteractiveHandler),
		wish.WithAddress(cfg.SSH.ListenAddr),
		wish.WithHostKeyPath(cfg.SSH.KeyPath),
		wish.WithMiddleware(mw...),
	)
	if err != nil {
		return nil, err
	}

	// Custom ServerConfig
	// which handles "none" auth method
	s.srv.ServerConfigCallback = s.ServerConfigCallback

	if cfg.SSH.MaxTimeout > 0 {
		s.srv.MaxTimeout = time.Duration(cfg.SSH.MaxTimeout) * time.Second
	}

	if cfg.SSH.IdleTimeout > 0 {
		s.srv.IdleTimeout = time.Duration(cfg.SSH.IdleTimeout) * time.Second
	}

	// Create client ssh key
	if _, err := os.Stat(cfg.SSH.ClientKeyPath); err != nil && os.IsNotExist(err) {
		_, err := keygen.New(cfg.SSH.ClientKeyPath, keygen.WithKeyType(keygen.Ed25519), keygen.WithWrite())
		if err != nil {
			return nil, fmt.Errorf("client ssh key: %w", err)
		}
	}

	return s, nil
}

// ListenAndServe starts the SSH server.
func (s *SSHServer) ListenAndServe() error {
	return s.srv.ListenAndServe()
}

// Serve starts the SSH server on the given net.Listener.
func (s *SSHServer) Serve(l net.Listener) error {
	return s.srv.Serve(l)
}

// Close closes the SSH server.
func (s *SSHServer) Close() error {
	return s.srv.Close()
}

// Shutdown gracefully shuts down the SSH server.
func (s *SSHServer) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

// PublicKeyAuthHandler handles public key authentication.
func (s *SSHServer) PublicKeyHandler(ctx ssh.Context, pk ssh.PublicKey) (allowed bool) {
	if pk == nil {
		return false
	}

	ak := backend.MarshalAuthorizedKey(pk)
	defer func(allowed *bool) {
		publicKeyCounter.WithLabelValues(ak, ctx.User(), strconv.FormatBool(*allowed)).Inc()
	}(&allowed)

	ac := s.cfg.Backend.AccessLevelByPublicKey("", pk)
	s.logger.Debugf("access level for %q: %s", ak, ac)
	allowed = ac >= backend.ReadWriteAccess
	return
}

// KeyboardInteractiveHandler handles keyboard interactive authentication.
// This is used after all public key authentication has failed.
func (s *SSHServer) KeyboardInteractiveHandler(ctx ssh.Context, _ gossh.KeyboardInteractiveChallenge) bool {
	ac := s.cfg.Backend.AllowKeyless()
	keyboardInteractiveCounter.WithLabelValues(ctx.User(), strconv.FormatBool(ac)).Inc()
	return ac
}

// NoClientAuthCallback handles "none" auth method.
func (s *SSHServer) NoClientAuthCallback(meta gossh.ConnMetadata) (*gossh.Permissions, error) {
	ac := s.cfg.Backend.AllowKeyless()
	keyboardInteractiveCounter.WithLabelValues(meta.User(), strconv.FormatBool(ac)).Inc()
	perms := &gossh.Permissions{}
	if !ac {
		return perms, fmt.Errorf("keyless access not allowed")
	}

	return perms, nil
}

// ServerConfigCallback returns a custom ssh.ServerConfig.
func (s *SSHServer) ServerConfigCallback(ctx ssh.Context) *gossh.ServerConfig {
	return &gossh.ServerConfig{
		NoClientAuth:         true,
		NoClientAuthCallback: s.NoClientAuthCallback,
	}
}

// Middleware adds Git server functionality to the ssh.Server. Repos are stored
// in the specified repo directory. The provided Hooks implementation will be
// checked for access on a per repo basis for a ssh.Session public key.
// Hooks.Push and Hooks.Fetch will be called on successful completion of
// their commands.
func (ss *SSHServer) Middleware(cfg *config.Config) wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			func() {
				cmd := s.Command()
				ctx := s.Context()
				be := ss.be.WithContext(ctx)
				if len(cmd) >= 2 && strings.HasPrefix(cmd[0], "git") {
					gc := cmd[0]
					// repo should be in the form of "repo.git"
					name := utils.SanitizeRepo(cmd[1])
					pk := s.PublicKey()
					ak := backend.MarshalAuthorizedKey(pk)
					access := cfg.Backend.AccessLevelByPublicKey(name, pk)
					// git bare repositories should end in ".git"
					// https://git-scm.com/docs/gitrepository-layout
					repo := name + ".git"
					reposDir := filepath.Join(cfg.DataPath, "repos")
					if err := git.EnsureWithin(reposDir, repo); err != nil {
						sshFatal(s, err)
						return
					}

					// Environment variables to pass down to git hooks.
					envs := []string{
						"SOFT_SERVE_REPO_NAME=" + name,
						"SOFT_SERVE_REPO_PATH=" + filepath.Join(reposDir, repo),
						"SOFT_SERVE_PUBLIC_KEY=" + ak,
					}

					ss.logger.Debug("git middleware", "cmd", gc, "access", access.String())
					repoDir := filepath.Join(reposDir, repo)
					switch gc {
					case git.ReceivePackBin:
						if access < backend.ReadWriteAccess {
							sshFatal(s, git.ErrNotAuthed)
							return
						}
						if _, err := be.Repository(name); err != nil {
							if _, err := be.CreateRepository(name, backend.RepositoryOptions{Private: false}); err != nil {
								log.Errorf("failed to create repo: %s", err)
								sshFatal(s, err)
								return
							}
							createRepoCounter.WithLabelValues(ak, s.User(), name).Inc()
						}
						if err := git.ReceivePack(s.Context(), s, s, s.Stderr(), repoDir, envs...); err != nil {
							sshFatal(s, git.ErrSystemMalfunction)
						}
						receivePackCounter.WithLabelValues(ak, s.User(), name).Inc()
						return
					case git.UploadPackBin, git.UploadArchiveBin:
						if access < backend.ReadOnlyAccess {
							sshFatal(s, git.ErrNotAuthed)
							return
						}

						gitPack := git.UploadPack
						counter := uploadPackCounter
						if gc == git.UploadArchiveBin {
							gitPack = git.UploadArchive
							counter = uploadArchiveCounter
						}

						err := gitPack(ctx, s, s, s.Stderr(), repoDir, envs...)
						if errors.Is(err, git.ErrInvalidRepo) {
							sshFatal(s, git.ErrInvalidRepo)
						} else if err != nil {
							sshFatal(s, git.ErrSystemMalfunction)
						}

						counter.WithLabelValues(ak, s.User(), name).Inc()
					}
				}
			}()
			sh(s)
		}
	}
}

// sshFatal prints to the session's STDOUT as a git response and exit 1.
func sshFatal(s ssh.Session, v ...interface{}) {
	git.WritePktline(s, v...)
	s.Exit(1) // nolint: errcheck
}
