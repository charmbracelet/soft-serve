package server

import (
	"context"
	"fmt"
	"log"
	"net"

	appCfg "github.com/charmbracelet/soft-serve/config"
	cm "github.com/charmbracelet/soft-serve/server/cmd"
	"github.com/charmbracelet/soft-serve/server/config"
	gm "github.com/charmbracelet/soft-serve/server/git"
	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	lm "github.com/charmbracelet/wish/logging"
	rm "github.com/charmbracelet/wish/recover"
	"github.com/gliderlabs/ssh"
	"github.com/muesli/termenv"
)

// Server is the Soft Serve server.
type Server struct {
	SSHServer *ssh.Server
	Config    *config.Config
	config    *appCfg.Config
}

// NewServer returns a new *ssh.Server configured to serve Soft Serve. The SSH
// server key-pair will be created if none exists. An initial admin SSH public
// key can be provided with authKey. If authKey is provided, access will be
// restricted to that key. If authKey is not provided, the server will be
// publicly writable until configured otherwise by cloning the `config` repo.
func NewServer(cfg *config.Config) *Server {
	ac, err := appCfg.NewConfig(cfg)
	if err != nil {
		log.Fatal(err)
	}
	mw := []wish.Middleware{
		rm.MiddlewareWithLogger(
			cfg.ErrorLog,
			// BubbleTea middleware.
			bm.MiddlewareWithProgramHandler(SessionHandler(ac), termenv.ANSI256),
			// Command middleware must come after the git middleware.
			cm.Middleware(ac),
			// Git middleware.
			gm.Middleware(cfg.RepoPath, ac),
			// Logging middleware must be last to be executed first.
			lm.Middleware(),
		),
	}
	s, err := wish.NewServer(
		ssh.PublicKeyAuth(ac.PublicKeyHandler),
		ssh.KeyboardInteractiveAuth(ac.KeyboardInteractiveHandler),
		wish.WithAddress(fmt.Sprintf("%s:%d", cfg.BindAddr, cfg.Port)),
		wish.WithHostKeyPath(cfg.KeyPath),
		wish.WithMiddleware(mw...),
	)
	if err != nil {
		log.Fatalln(err)
	}
	return &Server{
		SSHServer: s,
		Config:    cfg,
		config:    ac,
	}
}

// Reload reloads the server configuration.
func (srv *Server) Reload() error {
	return srv.config.Reload()
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
