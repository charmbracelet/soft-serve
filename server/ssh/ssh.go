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
	"github.com/charmbracelet/soft-serve/server/db"
	"github.com/charmbracelet/soft-serve/server/proto"
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
	dbx := db.FromContext(ctx)
	datastore := store.FromContext(ctx)
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
			ContextMiddleware(cfg, dbx, datastore, be, logger),
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

	defer func(allowed *bool) {
		publicKeyCounter.WithLabelValues(strconv.FormatBool(*allowed)).Inc()
	}(&allowed)

	user, _ := s.be.UserByPublicKey(ctx, pk)
	if user != nil {
		ctx.SetValue(proto.ContextKeyUser, user)
		allowed = true
	}

	return
}

// KeyboardInteractiveHandler handles keyboard interactive authentication.
// This is used after all public key authentication has failed.
func (s *SSHServer) KeyboardInteractiveHandler(ctx ssh.Context, _ gossh.KeyboardInteractiveChallenge) bool {
	ac := s.be.AllowKeyless(ctx)
	keyboardInteractiveCounter.WithLabelValues(strconv.FormatBool(ac)).Inc()
	return ac
}
