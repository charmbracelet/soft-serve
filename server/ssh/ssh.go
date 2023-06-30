package ssh

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/charmbracelet/keygen"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/server/access"
	"github.com/charmbracelet/soft-serve/server/auth"
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/creack/pty"

	// cm "github.com/charmbracelet/soft-serve/server/cmd"

	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/git"
	"github.com/charmbracelet/soft-serve/server/sshutils"
	"github.com/charmbracelet/soft-serve/server/store"
	"github.com/charmbracelet/soft-serve/server/utils"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	lm "github.com/charmbracelet/wish/logging"
	rm "github.com/charmbracelet/wish/recover"
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
	be     *backend.Backend
	ctx    context.Context
	logger *log.Logger
}

func setWinsize(f *os.File, w, h int) {
	syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCSWINSZ),
		uintptr(unsafe.Pointer(&struct{ h, w, x, y uint16 }{uint16(h), uint16(w), 0, 0})))
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
			// bm.MiddlewareWithProgramHandler(SessionHandler(ctx), termenv.ANSI256),
			// CLI middleware.
			// cm.Middleware(ctx, logger),
			// Git middleware.
			// s.Middleware(cfg),
			func(h ssh.Handler) ssh.Handler {
				return func(s ssh.Session) {
					ptyReq, winCh, isPty := s.Pty()
					cmds := s.Command()

					exe, err := os.Executable()
					if err != nil {
						s.Exit(1)
						return
					}

					cmd := exec.Command(exe, cmds...)
					if isPty {
						cmd.Env = append(cmd.Env, fmt.Sprintf("TERM=%s", ptyReq.Term))
					}
					cmd.Env = append(cmd.Env, fmt.Sprintf("SSH_ORIGINAL_COMMAND=%s", strings.Join(cmds, " ")))
					cmd.Env = append(cmd.Env, cfg.Environ()...)

					ptyf, tty, err := pty.Open()
					if err != nil {
						os.Exit(1)
						return
					}
					defer tty.Close()

					cmd.Env = append(cmd.Env, fmt.Sprintf("SSH_TTY=%s", tty.Name()))

					if cmd.Stdout == nil {
						cmd.Stdout = tty
					}
					if cmd.Stderr == nil {
						cmd.Stderr = tty
					}
					if cmd.Stdin == nil {
						cmd.Stdin = tty
					}

					cmd.SysProcAttr = &syscall.SysProcAttr{
						Setsid:  true,
						Setctty: true,
					}

					if err := cmd.Start(); err != nil {
						_ = ptyf.Close()
						os.Exit(1)
						return
					}
					go func() {
						for win := range winCh {
							setWinsize(ptyf, win.Width, win.Height)
						}
					}()
					go func() {
						io.Copy(ptyf, s) // stdin
					}()
					io.Copy(s, ptyf) // stdout

					cmd.Wait()
					h(s)
				}
			},
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
	ctx.SetValue(config.ContextKeyConfig, s.cfg)
	ctx.SetValue(ssh.ContextKeyPublicKey, pk)

	if pk == nil {
		return false
	}

	var ac access.AccessLevel
	var user auth.User
	ak := sshutils.MarshalAuthorizedKey(pk)

	defer func(allowed *bool) {
		publicKeyCounter.WithLabelValues(ak, ctx.User(), strconv.FormatBool(*allowed)).Inc()
		s.logger.Debugf("access level for %q: %s", ak, ac)
		ctx.SetValue(auth.ContextKeyUser, user)
	}(&allowed)

	user, _ = s.be.Authenticate(ctx, auth.NewPublicKey(pk))
	ac, _ = s.be.AccessLevel(ctx, "", user)
	allowed = ac >= access.ReadWriteAccess
	return
}

// KeyboardInteractiveHandler handles keyboard interactive authentication.
// This is used after all public key authentication has failed.
func (s *SSHServer) KeyboardInteractiveHandler(ctx ssh.Context, _ gossh.KeyboardInteractiveChallenge) bool {
	ctx.SetValue(config.ContextKeyConfig, s.cfg)
	ac := s.be.AllowKeyless(ctx)
	keyboardInteractiveCounter.WithLabelValues(ctx.User(), strconv.FormatBool(ac)).Inc()
	return ac
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
				cmdLine := s.Command()
				ctx := s.Context()

				if len(cmdLine) >= 2 && strings.HasPrefix(cmdLine[0], "git") {
					// repo should be in the form of "repo.git"
					name := utils.SanitizeRepo(cmdLine[1])
					pk := s.PublicKey()
					ak := sshutils.MarshalAuthorizedKey(pk)
					user, _ := ss.be.Authenticate(ctx, auth.NewPublicKey(pk))
					ac, _ := ss.be.AccessLevel(ctx, name, user)

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
						"SOFT_SERVE_USERNAME=" + ctx.User(),
					}

					// Add ssh session & config environ
					envs = append(envs, s.Environ()...)
					envs = append(envs, cfg.Environ()...)

					repoDir := filepath.Join(reposDir, repo)
					service := git.Service(cmdLine[0])
					cmd := git.ServiceCommand{
						Stdin:  s,
						Stdout: s,
						Stderr: s.Stderr(),
						Env:    envs,
						Dir:    repoDir,
					}

					ss.logger.Debug("git middleware", "cmd", service, "access", ac.String())

					switch service {
					case git.ReceivePackService:
						if ac < access.ReadWriteAccess {
							sshFatal(s, git.ErrUnauthorized)
							return
						}
						if _, err := ss.be.Repository(ctx, name); err != nil {
							if _, err := ss.be.CreateRepository(ctx, name, store.RepositoryOptions{Private: false}); err != nil {
								log.Errorf("failed to create repo: %s", err)
								sshFatal(s, err)
								return
							}

							createRepoCounter.WithLabelValues(ak, s.User(), name).Inc()
						}

						if err := git.ReceivePack(ctx, cmd); err != nil {
							sshFatal(s, git.ErrSystemMalfunction)
						}

						if err := git.EnsureDefaultBranch(ctx, cmd); err != nil {
							sshFatal(s, git.ErrSystemMalfunction)
						}

						receivePackCounter.WithLabelValues(ak, s.User(), name).Inc()
						return
					case git.UploadPackService, git.UploadArchiveService:
						if ac < access.ReadOnlyAccess {
							sshFatal(s, git.ErrUnauthorized)
							return
						}

						handler := git.UploadPack
						counter := uploadPackCounter
						if service == git.UploadArchiveService {
							handler = git.UploadArchive
							counter = uploadArchiveCounter
						}

						err := handler(ctx, cmd)
						if errors.Is(err, git.ErrNotExist) {
							sshFatal(s, git.ErrNotExist)
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
