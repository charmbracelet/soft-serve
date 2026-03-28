package serve

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"

	"charm.land/log/v2"

	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/cron"
	"github.com/charmbracelet/soft-serve/pkg/daemon"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/jobs"
	sshsrv "github.com/charmbracelet/soft-serve/pkg/ssh"
	"github.com/charmbracelet/soft-serve/pkg/stats"
	"github.com/charmbracelet/soft-serve/pkg/web"
	"github.com/charmbracelet/ssh"
	"golang.org/x/sync/errgroup"
)

// Server is the Soft Serve server.
type Server struct {
	SSHServer   *sshsrv.SSHServer
	GitDaemon   *daemon.GitDaemon
	HTTPServer  *web.HTTPServer
	StatsServer *stats.StatsServer
	CertLoader  *CertReloader
	Cron        *cron.Scheduler
	Config      *config.Config
	Backend     *backend.Backend
	DB          *db.DB

	logger *log.Logger
	ctx    context.Context
}

// NewServer returns a new *Server configured to serve Soft Serve. The SSH
// server key-pair will be created if none exists.
// It expects a context with *backend.Backend, *db.DB, *log.Logger, and
// *config.Config attached.
func NewServer(ctx context.Context) (*Server, error) {
	var err error
	cfg := config.FromContext(ctx)
	be := backend.FromContext(ctx)
	db := db.FromContext(ctx)
	logger := log.FromContext(ctx).WithPrefix("server")
	srv := &Server{
		Config:  cfg,
		Backend: be,
		DB:      db,
		logger:  log.FromContext(ctx).WithPrefix("server"),
		ctx:     ctx,
	}

	// Add cron jobs.
	sched := cron.NewScheduler(ctx)
	for n, j := range jobs.List() {
		id, err := sched.AddFunc(j.Runner.Spec(ctx), j.Runner.Func(ctx))
		if err != nil {
			logger.Warn("error adding cron job", "job", n, "err", err)
		}

		j.ID = id
	}

	srv.Cron = sched

	srv.SSHServer, err = sshsrv.NewSSHServer(ctx)
	if err != nil {
		return nil, fmt.Errorf("create ssh server: %w", err)
	}

	srv.GitDaemon, err = daemon.NewGitDaemon(ctx)
	if err != nil {
		return nil, fmt.Errorf("create git daemon: %w", err)
	}

	srv.HTTPServer, err = web.NewHTTPServer(ctx)
	if err != nil {
		return nil, fmt.Errorf("create http server: %w", err)
	}

	srv.StatsServer, err = stats.NewStatsServer(ctx)
	if err != nil {
		return nil, fmt.Errorf("create stats server: %w", err)
	}

	if cfg.HTTP.TLSKeyPath != "" && cfg.HTTP.TLSCertPath != "" {
		srv.CertLoader, err = NewCertReloader(cfg.HTTP.TLSCertPath, cfg.HTTP.TLSKeyPath, logger)
		if err != nil {
			return nil, fmt.Errorf("create cert reloader: %w", err)
		}

		srv.HTTPServer.SetTLSConfig(&tls.Config{
			GetCertificate: srv.CertLoader.GetCertificateFunc(),
		})
	}

	return srv, nil
}

// ReloadCertificates reloads the TLS certificates for the HTTP server.
func (s *Server) ReloadCertificates() error {
	if s.CertLoader == nil {
		return nil
	}
	return s.CertLoader.Reload()
}

// Start starts the SSH server.
func (s *Server) Start() error {
	// Pre-bind all listeners before launching goroutines so that port
	// permission errors (e.g. EACCES on privileged ports) are returned
	// immediately instead of being swallowed by a blocked errgroup.Wait().
	var sshListener, gitListener, httpListener, statsListener net.Listener
	var err error

	if s.Config.SSH.Enabled {
		sshListener, err = net.Listen("tcp", s.Config.SSH.ListenAddr)
		if err != nil {
			return fmt.Errorf("listen ssh %s: %w", s.Config.SSH.ListenAddr, err)
		}
	}

	if s.Config.Git.Enabled {
		gitListener, err = net.Listen("tcp", s.Config.Git.ListenAddr)
		if err != nil {
			return fmt.Errorf("listen git daemon %s: %w", s.Config.Git.ListenAddr, err)
		}
	}

	if s.Config.HTTP.Enabled {
		httpListener, err = net.Listen("tcp", s.Config.HTTP.ListenAddr)
		if err != nil {
			return fmt.Errorf("listen http %s: %w", s.Config.HTTP.ListenAddr, err)
		}
	}

	if s.Config.Stats.Enabled {
		statsListener, err = net.Listen("tcp", s.Config.Stats.ListenAddr)
		if err != nil {
			return fmt.Errorf("listen stats %s: %w", s.Config.Stats.ListenAddr, err)
		}
	}

	errg, _ := errgroup.WithContext(s.ctx)

	if s.Config.SSH.Enabled {
		errg.Go(func() error {
			s.logger.Print("Starting SSH server", "addr", s.Config.SSH.ListenAddr)
			if err := s.SSHServer.Serve(sshListener); !errors.Is(err, ssh.ErrServerClosed) {
				return err
			}
			return nil
		})
	}

	if s.Config.Git.Enabled {
		errg.Go(func() error {
			s.logger.Print("Starting Git daemon", "addr", s.Config.Git.ListenAddr)
			if err := s.GitDaemon.Serve(gitListener); !errors.Is(err, daemon.ErrServerClosed) {
				return err
			}
			return nil
		})
	}

	if s.Config.HTTP.Enabled {
		errg.Go(func() error {
			s.logger.Print("Starting HTTP server", "addr", s.Config.HTTP.ListenAddr)
			if err := s.HTTPServer.Serve(httpListener); !errors.Is(err, http.ErrServerClosed) {
				return err
			}
			return nil
		})
	}

	if s.Config.Stats.Enabled {
		errg.Go(func() error {
			s.logger.Print("Starting Stats server", "addr", s.Config.Stats.ListenAddr)
			if err := s.StatsServer.Serve(statsListener); !errors.Is(err, http.ErrServerClosed) {
				return err
			}
			return nil
		})
	}

	errg.Go(func() error {
		s.Cron.Start()
		return nil
	})
	return errg.Wait()
}

// Shutdown lets the server gracefully shutdown.
func (s *Server) Shutdown(ctx context.Context) error {
	errg, ctx := errgroup.WithContext(ctx)
	errg.Go(func() error {
		return s.GitDaemon.Shutdown(ctx)
	})
	errg.Go(func() error {
		return s.HTTPServer.Shutdown(ctx)
	})
	errg.Go(func() error {
		return s.SSHServer.Shutdown(ctx)
	})
	errg.Go(func() error {
		return s.StatsServer.Shutdown(ctx)
	})
	errg.Go(func() error {
		for _, j := range jobs.List() {
			s.Cron.Remove(j.ID)
		}
		s.Cron.Stop()
		return nil
	})
	// defer s.DB.Close() // nolint: errcheck
	return errg.Wait()
}

// Close closes the SSH server.
func (s *Server) Close() error {
	var errg errgroup.Group
	errg.Go(s.GitDaemon.Close)
	errg.Go(s.HTTPServer.Close)
	errg.Go(s.SSHServer.Close)
	errg.Go(s.StatsServer.Close)
	errg.Go(func() error {
		s.Cron.Stop()
		return nil
	})
	// defer s.DB.Close() // nolint: errcheck
	return errg.Wait()
}
