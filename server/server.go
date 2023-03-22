package server

import (
	"context"

	"github.com/charmbracelet/log"

	"github.com/charmbracelet/soft-serve/server/backend"
	cm "github.com/charmbracelet/soft-serve/server/cmd"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/git/daemon"
	gm "github.com/charmbracelet/soft-serve/server/git/ssh"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	lm "github.com/charmbracelet/wish/logging"
	rm "github.com/charmbracelet/wish/recover"
	"github.com/muesli/termenv"
	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/sync/errgroup"
)

// Server is the Soft Serve server.
type Server struct {
	SSHServer *ssh.Server
	GitDaemon *daemon.Daemon
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

	var err error
	srv.SSHServer, err = wish.NewServer(
		ssh.PublicKeyAuth(srv.PublicKeyHandler),
		ssh.KeyboardInteractiveAuth(srv.KeyboardInteractiveHandler),
		wish.WithAddress(cfg.SSH.ListenAddr),
		wish.WithHostKeyPath(cfg.SSH.KeyPath),
		wish.WithMiddleware(mw...),
	)
	if err != nil {
		return nil, err
	}

	srv.GitDaemon, err = daemon.NewDaemon(cfg)
	if err != nil {
		return nil, err
	}

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
func (s *Server) Start() error {
	var errg errgroup.Group
	errg.Go(func() error {
		log.Print("Starting Git daemon", "addr", s.Config.Git.ListenAddr)
		if err := s.GitDaemon.Start(); err != daemon.ErrServerClosed {
			return err
		}
		return nil
	})
	errg.Go(func() error {
		log.Print("Starting SSH server", "addr", s.Config.SSH.ListenAddr)
		if err := s.SSHServer.ListenAndServe(); err != ssh.ErrServerClosed {
			return err
		}
		return nil
	})
	return errg.Wait()
}

// Shutdown lets the server gracefully shutdown.
func (s *Server) Shutdown(ctx context.Context) error {
	var errg errgroup.Group
	errg.Go(func() error {
		return s.GitDaemon.Shutdown(ctx)
	})
	errg.Go(func() error {
		return s.SSHServer.Shutdown(ctx)
	})
	return errg.Wait()
}

// Close closes the SSH server.
func (s *Server) Close() error {
	var errg errgroup.Group
	errg.Go(func() error {
		return s.SSHServer.Close()
	})
	errg.Go(func() error {
		return s.GitDaemon.Close()
	})
	return errg.Wait()
}
