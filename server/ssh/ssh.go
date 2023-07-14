package ssh

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/charmbracelet/keygen"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/git"
	"github.com/charmbracelet/soft-serve/server/sshutils"
	"github.com/charmbracelet/soft-serve/server/store"
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
	}, []string{"allowed"})

	keyboardInteractiveCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "soft_serve",
		Subsystem: "ssh",
		Name:      "keyboard_interactive_auth_total",
		Help:      "The total number of keyboard interactive auth requests",
	}, []string{"allowed"})

	uploadPackCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "soft_serve",
		Subsystem: "git",
		Name:      "upload_pack_total",
		Help:      "The total number of git-upload-pack requests",
	}, []string{"repo"})

	receivePackCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "soft_serve",
		Subsystem: "git",
		Name:      "receive_pack_total",
		Help:      "The total number of git-receive-pack requests",
	}, []string{"repo"})

	uploadArchiveCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "soft_serve",
		Subsystem: "git",
		Name:      "upload_archive_total",
		Help:      "The total number of git-upload-archive requests",
	}, []string{"repo"})

	uploadPackSeconds = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "soft_serve",
		Subsystem: "git",
		Name:      "upload_pack_seconds_total",
		Help:      "The total time spent on git-upload-pack requests",
	}, []string{"repo"})

	receivePackSeconds = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "soft_serve",
		Subsystem: "git",
		Name:      "receive_pack_seconds_total",
		Help:      "The total time spent on git-receive-pack requests",
	}, []string{"repo"})

	uploadArchiveSeconds = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "soft_serve",
		Subsystem: "git",
		Name:      "upload_archive_seconds_total",
		Help:      "The total time spent on git-upload-archive requests",
	}, []string{"repo"})

	createRepoCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "soft_serve",
		Subsystem: "ssh",
		Name:      "create_repo_total",
		Help:      "The total number of create repo requests",
	}, []string{"repo"})
)

// SSHServer is a SSH server that implements the git protocol.
type SSHServer struct { // nolint: revive
	srv    *ssh.Server
	cfg    *config.Config
	be     *backend.Backend
	ctx    context.Context
	logger *log.Logger
}

// NewSSHServer returns a new SSHServer.
func NewSSHServer(ctx context.Context) (*SSHServer, error) {
	cfg := config.FromContext(ctx)
	logger := log.FromContext(ctx).WithPrefix("ssh")
	be := backend.FromContext(ctx)

	var err error
	s := &SSHServer{
		cfg:    cfg,
		ctx:    ctx,
		be:     be,
		logger: logger,
	}

	mw := []wish.Middleware{
		rm.MiddlewareWithLogger(
			logger,
			// BubbleTea middleware.
			bm.MiddlewareWithProgramHandler(SessionHandler, termenv.ANSI256),
			// CLI middleware.
			CommandMiddleware,
			// Context middleware.
			ContextMiddleware(cfg, be, logger),
			// Logging middleware.
			lm.MiddlewareWithLogger(
				&loggerAdapter{logger, log.DebugLevel},
			),
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
	if pk == nil {
		return false
	}

	ak := sshutils.MarshalAuthorizedKey(pk)
	defer func(allowed *bool) {
		publicKeyCounter.WithLabelValues(strconv.FormatBool(*allowed)).Inc()
	}(&allowed)

	ac := s.be.AccessLevelByPublicKey(ctx, "", pk)
	s.logger.Debugf("access level for %q: %s", ak, ac)
	allowed = ac >= store.ReadWriteAccess
	return
}

// KeyboardInteractiveHandler handles keyboard interactive authentication.
// This is used after all public key authentication has failed.
func (s *SSHServer) KeyboardInteractiveHandler(ctx ssh.Context, _ gossh.KeyboardInteractiveChallenge) bool {
	ac := s.be.AllowKeyless(ctx)
	keyboardInteractiveCounter.WithLabelValues(strconv.FormatBool(ac)).Inc()
	return ac
}

// sshFatal prints to the session's STDOUT as a git response and exit 1.
func sshFatal(s ssh.Session, err error) {
	git.WritePktlineErr(s, err) // nolint: errcheck
	s.Exit(1)                   // nolint: errcheck
}
