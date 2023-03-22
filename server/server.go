package server

import (
	"context"
	"net"

	"github.com/charmbracelet/log"

	"github.com/charmbracelet/soft-serve/server/backend"
	cm "github.com/charmbracelet/soft-serve/server/cmd"
	"github.com/charmbracelet/soft-serve/server/config"
	gm "github.com/charmbracelet/soft-serve/server/git/ssh"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	lm "github.com/charmbracelet/wish/logging"
	rm "github.com/charmbracelet/wish/recover"
	"github.com/muesli/termenv"
	gossh "golang.org/x/crypto/ssh"
)

// Server is the Soft Serve server.
type Server struct {
	SSHServer *ssh.Server
	Config    *config.Config
	Backend   backend.Backend
	Access    backend.AccessMethod
}

// NewServer returns a new *ssh.Server configured to serve Soft Serve. The SSH
// server key-pair will be created if none exists. An initial admin SSH public
// key can be provided with authKey. If authKey is provided, access will be
// restricted to that key. If authKey is not provided, the server will be
// publicly writable until configured otherwise by cloning the `config` repo.
func NewServer(cfg *config.Config) (*Server, error) {
	srv := &Server{
		Config:  cfg,
		Backend: cfg.Backend,
		Access:  cfg.Access,
	}
	mw := []wish.Middleware{
		rm.MiddlewareWithLogger(
			cfg.ErrorLog,
			// BubbleTea middleware.
			bm.MiddlewareWithProgramHandler(SessionHandler(cfg), termenv.ANSI256),
			// Command middleware must come after the git middleware.
			cm.Middleware(cfg),
			// Git middleware.
			gm.Middleware(cfg),
			lm.MiddlewareWithLogger(log.StandardLog(log.StandardLogOptions{ForceLevel: log.DebugLevel})),
		),
	}
	s, err := wish.NewServer(
		ssh.PublicKeyAuth(srv.PublicKeyHandler),
		ssh.KeyboardInteractiveAuth(srv.KeyboardInteractiveHandler),
		wish.WithAddress(cfg.SSH.ListenAddr),
		wish.WithHostKeyPath(cfg.KeyPath),
		wish.WithMiddleware(mw...),
	)
	if err != nil {
		return nil, err
	}
	srv.SSHServer = s
	return srv, nil
}

// PublicKeyAuthHandler handles public key authentication.
func (srv *Server) PublicKeyHandler(ctx ssh.Context, pk ssh.PublicKey) bool {
	return srv.Access.AccessLevel("", pk) > backend.NoAccess
}

// KeyboardInteractiveHandler handles keyboard interactive authentication.
func (srv *Server) KeyboardInteractiveHandler(_ ssh.Context, _ gossh.KeyboardInteractiveChallenge) bool {
	return true
}

// Start starts the SSH server.
func (srv *Server) Start() error {
	if err := srv.SSHServer.ListenAndServe(); err != ssh.ErrServerClosed {
		return err
	}
	return nil
}

// Serve serves the SSH server using the provided listener.
func (srv *Server) Serve(l net.Listener) error {
	if err := srv.SSHServer.Serve(l); err != ssh.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown lets the server gracefully shutdown.
func (srv *Server) Shutdown(ctx context.Context) error {
	return srv.SSHServer.Shutdown(ctx)
}

// Close closes the SSH server.
func (srv *Server) Close() error {
	return srv.SSHServer.Close()
}
