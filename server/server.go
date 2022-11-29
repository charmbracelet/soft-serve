package server

import (
	"context"
	"fmt"
	"log"
	"time"

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
	s := &Server{Config: cfg}
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
			cm.Middleware(cfg),
			// Git middleware.
			gm.Middleware(cfg.RepoPath(), ac),
			// Logging middleware must be last to be executed first.
			lm.Middleware(),
		),
	}

	opts := []ssh.Option{ssh.PublicKeyAuth(cfg.PublicKeyHandler)}
	if cfg.SSH.AllowKeyless {
		opts = append(opts, ssh.KeyboardInteractiveAuth(cfg.KeyboardInteractiveHandler))
	}
	if cfg.SSH.AllowPassword {
		opts = append(opts, ssh.PasswordAuth(cfg.PasswordHandler))
	}
	opts = append(opts,
		wish.WithAddress(fmt.Sprintf("%s:%d", cfg.Host, cfg.SSH.Port)),
		wish.WithHostKeyPath(cfg.PrivateKeyPath()),
		wish.WithMiddleware(mw...),
	)
	sh, err := wish.NewServer(opts...)
	if err != nil {
		log.Fatalln(err)
	}
	if cfg.SSH.MaxTimeout > 0 {
		sh.MaxTimeout = time.Duration(cfg.SSH.MaxTimeout) * time.Second
	}
	if cfg.SSH.IdleTimeout > 0 {
		sh.IdleTimeout = time.Duration(cfg.SSH.IdleTimeout) * time.Second
	}
	s.SSHServer = sh
	d, err := daemon.NewDaemon(cfg)
	if err != nil {
		log.Fatalln(err)
	}
	s.GitServer = d
	return s
}

// Reload reloads the server configuration.
func (s *Server) Reload() error {
	return s.config.Reload()
}

// Start starts the SSH server.
func (s *Server) Start() error {
	var errg errgroup.Group
	errg.Go(func() error {
		log.Printf("Starting Git server on %s:%d", s.Config.Host, s.Config.Git.Port)
		if err := s.GitServer.Start(); err != daemon.ErrServerClosed {
			return err
		}
		return nil
	})
	errg.Go(func() error {
		log.Printf("Starting SSH server on %s:%d", s.Config.Host, s.Config.SSH.Port)
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
