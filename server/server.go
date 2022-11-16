package server

import (
	"context"
	"fmt"
	"log"

	appCfg "github.com/charmbracelet/soft-serve/config"
	cm "github.com/charmbracelet/soft-serve/server/cmd"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/git/daemon"
	gm "github.com/charmbracelet/soft-serve/server/git/ssh"
	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	lm "github.com/charmbracelet/wish/logging"
	rm "github.com/charmbracelet/wish/recover"
	"github.com/gliderlabs/ssh"
	"github.com/muesli/termenv"
	"golang.org/x/sync/errgroup"
)

// Server is the Soft Serve server.
type Server struct {
	SSHServer *ssh.Server
	GitServer *daemon.Daemon
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
	d, err := daemon.NewDaemon(cfg, ac)
	if err != nil {
		log.Fatalln(err)
	}
	return &Server{
		SSHServer: s,
		GitServer: d,
		Config:    cfg,
		config:    ac,
	}
}

// Reload reloads the server configuration.
func (s *Server) Reload() error {
	return s.config.Reload()
}

// Start starts the SSH server.
func (s *Server) Start() error {
	var errg errgroup.Group
	errg.Go(func() error {
		log.Printf("Starting Git server on %s:%d", s.Config.BindAddr, s.Config.GitPort)
		if err := s.GitServer.Start(); err != daemon.ErrServerClosed {
			return err
		}
		return nil
	})
	errg.Go(func() error {
		log.Printf("Starting SSH server on %s:%d", s.Config.BindAddr, s.Config.Port)
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
		return s.SSHServer.Shutdown(ctx)
	})
	errg.Go(func() error {
		return s.GitServer.Shutdown(ctx)
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
		return s.GitServer.Close()
	})
	return errg.Wait()
}
