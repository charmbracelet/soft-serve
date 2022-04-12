package server

import (
	"context"
	"fmt"
	"log"
	"net"

	appCfg "github.com/charmbracelet/soft-serve/config"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	gm "github.com/charmbracelet/wish/git"
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
			softMiddleware(ac),
			bm.MiddlewareWithProgramHandler(SessionHandler(ac), termenv.ANSI256),
			gm.Middleware(cfg.RepoPath, ac),
			lm.Middleware(),
		),
	}
	s, err := wish.NewServer(
		ssh.PublicKeyAuth(ac.PublicKeyHandler),
		ssh.PasswordAuth(ac.PasswordHandler),
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
	return srv.SSHServer.ListenAndServe()
}

// Serve serves the SSH server using the provided listener.
func (srv *Server) Serve(l net.Listener) error {
	return srv.SSHServer.Serve(l)
}

// Shutdown lets the server gracefully shutdown.
func (srv *Server) Shutdown(ctx context.Context) error {
	return srv.SSHServer.Shutdown(ctx)
}

// Close closes the SSH server.
func (srv *Server) Close() error {
	return srv.SSHServer.Close()
}
