package server

import (
	"context"
	"fmt"
	pm "github.com/charmbracelet/promwish"
	"log"
	"net"
	"path/filepath"
	"strings"

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
			// Note: disable pushing to subdirectories as it can create
			// conflicts with existing repos. This only affects the git
			// middleware.
			//
			// This is related to
			// https://github.com/charmbracelet/soft-serve/issues/120
			// https://github.com/charmbracelet/wish/commit/8808de520d3ea21931f13113c6b0b6d0141272d4
			func(sh ssh.Handler) ssh.Handler {
				return func(s ssh.Session) {
					cmds := s.Command()
					if len(cmds) == 2 && strings.HasPrefix(cmds[0], "git") {
						repo := strings.TrimSuffix(strings.TrimPrefix(cmds[1], "/"), "/")
						repo = filepath.Clean(repo)
						if n := strings.Count(repo, "/"); n != 0 {
							wish.Fatalln(s, fmt.Errorf("invalid repo path: subdirectories not allowed"))
							return
						}
					}
					sh(s)
				}
			},
			lm.Middleware(),
			pm.Middleware("localhost:9222", "soft-serve"),
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
