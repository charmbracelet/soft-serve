package ssh

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"charm.land/log/v2"
	"charm.land/wish/v2"
	bm "charm.land/wish/v2/bubbletea"
	rm "charm.land/wish/v2/recover"
	"github.com/charmbracelet/keygen"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/store"
	"github.com/charmbracelet/ssh"
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

// tokenAuthUserIDKey is a package-private context key used to carry the
// token-authenticated user ID from KeyboardInteractiveHandler to
// AuthenticationMiddleware. Using a private type (not a string) prevents
// injection via SSH certificate extensions, which use string-keyed maps.
type tokenAuthUserIDKey struct{}

// SSHServer is a SSH server that implements the git protocol.
type SSHServer struct { //nolint: revive
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
			bm.MiddlewareWithProgramHandler(SessionHandler),
			// CLI middleware.
			CommandMiddleware,
			// Logging middleware.
			LoggingMiddleware,
			// Authentication middleware.
			// gossh.PublicKeyHandler doesn't guarantee that the public key
			// is in fact the one used for authentication, so we need to
			// check it again here.
			AuthenticationMiddleware,
			// Context middleware.
			// This must come first to set up the context.
			ContextMiddleware(cfg, dbx, datastore, be, logger),
		),
	}

	// Ensure the directory for the host key file exists.
	if dir := filepath.Dir(cfg.SSH.KeyPath); dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("create ssh key dir: %w", err)
		}
	}

	opts := []ssh.Option{
		ssh.PublicKeyAuth(s.PublicKeyHandler),
		ssh.KeyboardInteractiveAuth(s.KeyboardInteractiveHandler),
		wish.WithAddress(cfg.SSH.ListenAddr),
		wish.WithHostKeyPath(cfg.SSH.KeyPath),
		wish.WithMiddleware(mw...),
	}

	// TODO: Support a real PTY in future version.
	opts = append(opts, ssh.EmulatePty())

	s.srv, err = wish.NewServer(opts...)
	if err != nil {
		return nil, err
	}

	if config.IsDebug() {
		s.srv.ServerConfigCallback = func(_ ssh.Context) *gossh.ServerConfig {
			return &gossh.ServerConfig{
				AuthLogCallback: func(conn gossh.ConnMetadata, method string, err error) {
					logger.Debug("authentication", "user", conn.User(), "method", method, "err", err)
				},
			}
		}
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

func initializePermissions(ctx ssh.Context) {
	perms := ctx.Permissions()
	if perms == nil || perms.Permissions == nil {
		perms = &ssh.Permissions{Permissions: &gossh.Permissions{}}
	}
	if perms.Extensions == nil {
		perms.Extensions = make(map[string]string)
	}
	ctx.SetValue(ssh.ContextKeyPermissions, perms)
}

// PublicKeyHandler handles public key authentication.
func (s *SSHServer) PublicKeyHandler(ctx ssh.Context, pk ssh.PublicKey) (allowed bool) {
	if pk == nil {
		return false
	}

	allowed = true

	// XXX: store the first "approved" public-key fingerprint in the
	// permissions block to use for authentication later.
	initializePermissions(ctx)
	perms := ctx.Permissions()

	// Only record the first offered key — the SSH client will use this
	// key for the authenticated signature step. Overwriting on each probe
	// would store the LAST key offered, which may differ from the one the
	// client ultimately signs with, causing a fingerprint mismatch in
	// AuthenticationMiddleware.
	if perms.Extensions["pubkey-fp"] == "" {
		perms.Extensions["pubkey-fp"] = gossh.FingerprintSHA256(pk)
		ctx.SetValue(ssh.ContextKeyPermissions, perms)
	}

	return
}

// KeyboardInteractiveHandler handles keyboard interactive authentication.
// It prompts for an access token and validates it. If no valid token is
// provided, it falls back to AllowKeyless behavior.
func (s *SSHServer) KeyboardInteractiveHandler(ctx ssh.Context, challenge gossh.KeyboardInteractiveChallenge) bool {
	initializePermissions(ctx)
	perms := ctx.Permissions()

	// Prompt the user for an access token.
	answers, err := challenge("", "", []string{"Access Token: "}, []bool{false})
	if err != nil {
		s.logger.Debug("keyboard-interactive challenge failed", "err", err)
	} else if len(answers) > 0 && answers[0] != "" {
		token := answers[0]
		user, tokenErr := s.be.UserByAccessToken(ctx, token)
		if tokenErr == nil && user != nil {
			// Valid token: store the user ID via a package-private context key.
			// We intentionally do NOT use perms.Extensions here — certificate
			// extensions from gossh are merged into the same map, so a string
			// key can be injected by a client presenting a crafted certificate.
			//
			// Clear pubkey-fp so AuthenticationMiddleware's fingerprint guard
			// does not reject this keyless token-auth session.
			perms.Extensions["pubkey-fp"] = ""
			ctx.SetValue(ssh.ContextKeyPermissions, perms)
			ctx.SetValue(tokenAuthUserIDKey{}, user.ID())
			keyboardInteractiveCounter.WithLabelValues("true").Inc()
			s.logger.Info("keyboard-interactive token auth succeeded", "username", user.Username())
			return true
		}
		if tokenErr != nil {
			s.logger.Warn("keyboard-interactive token auth failed", "err", tokenErr)
		} else {
			s.logger.Warn("keyboard-interactive token auth failed", "err", "user not found")
		}
	}

	// No valid token: fall back to AllowKeyless behavior.
	ac := s.be.AllowKeyless(ctx)
	keyboardInteractiveCounter.WithLabelValues(strconv.FormatBool(ac)).Inc()

	if ac {
		perms.Extensions["pubkey-fp"] = ""
	}
	return ac
}
