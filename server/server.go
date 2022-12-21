package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

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
	SSHServer  *ssh.Server
	GitServer  *daemon.Daemon
	HTTPServer *http.Server
	Config     *config.Config
}

// NewServer returns a new *ssh.Server configured to serve Soft Serve. The SSH
// server key-pair will be created if none exists. An initial admin SSH public
// key can be provided with authKey. If authKey is provided, access will be
// restricted to that key. If authKey is not provided, the server will be
// publicly writable until configured otherwise by cloning the `config` repo.
func NewServer(cfg *config.Config) *Server {
	s := &Server{Config: cfg}
	mw := []wish.Middleware{
		rm.MiddlewareWithLogger(
			log.Default(),
			// BubbleTea middleware.
			bm.MiddlewareWithProgramHandler(SessionHandler(cfg), termenv.ANSI256),
			// Command middleware must come after the git middleware.
			cm.Middleware(cfg),
			// Git middleware.
			gm.Middleware(cfg.RepoPath(), cfg),
			// Logging middleware must be last to be executed first.
			lm.Middleware(),
		),
	}

	opts := []ssh.Option{
		wish.WithAddress(fmt.Sprintf("%s:%d", cfg.Host, cfg.SSH.Port)),
		wish.WithPublicKeyAuth(cfg.PublicKeyHandler),
		wish.WithMiddleware(mw...),
	}
	if cfg.SSH.AllowKeyless {
		opts = append(opts, ssh.KeyboardInteractiveAuth(cfg.KeyboardInteractiveHandler))
	}
	if cfg.SSH.AllowPassword {
		opts = append(opts, ssh.PasswordAuth(cfg.PasswordHandler))
	}
	if cfg.SSH.Key != "" {
		opts = append(opts, wish.WithHostKeyPEM([]byte(cfg.SSH.Key)))
	} else {
		opts = append(opts, wish.WithHostKeyPath(cfg.PrivateKeyPath()))
	}
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
	if cfg.Git.Enabled {
		d, err := daemon.NewDaemon(cfg)
		if err != nil {
			log.Fatalln(err)
		}
		s.GitServer = d
	}
	if cfg.HTTP.Enabled {
		s.HTTPServer = newHTTPServer(cfg)
	}
	return s
}

// Start starts the SSH server.
func (s *Server) Start() error {
	var errg errgroup.Group
	if s.Config.Git.Enabled {
		errg.Go(func() error {
			log.Printf("Starting Git server on %s:%d", s.Config.Host, s.Config.Git.Port)
			if err := s.GitServer.Start(); err != daemon.ErrServerClosed {
				return err
			}
			return nil
		})
	}
	if s.Config.HTTP.Enabled {
		errg.Go(func() error {
			log.Printf("Starting HTTP server on %s:%d", s.Config.Host, s.Config.HTTP.Port)
			if err := s.HTTPServer.ListenAndServe(); err != http.ErrServerClosed {
				return err
			}
			return nil
		})
	}
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
	if s.Config.Git.Enabled {
		errg.Go(func() error {
			return s.GitServer.Shutdown(ctx)
		})
	}
	if s.Config.HTTP.Enabled {
		errg.Go(func() error {
			return s.HTTPServer.Shutdown(ctx)
		})
	}
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
	if s.Config.Git.Enabled {
		errg.Go(func() error {
			return s.GitServer.Close()
		})
	}
	if s.Config.HTTP.Enabled {
		errg.Go(func() error {
			return s.HTTPServer.Close()
		})
	}
	return errg.Wait()
}
