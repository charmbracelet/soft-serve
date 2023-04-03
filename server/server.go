package server

import (
	"context"
	"net/http"
	"path/filepath"

	"github.com/charmbracelet/keygen"
	"github.com/charmbracelet/log"

	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/backend/sqlite"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/cron"
	"github.com/charmbracelet/ssh"
	"golang.org/x/sync/errgroup"
)

var (
	logger = log.WithPrefix("server")
)

// Server is the Soft Serve server.
type Server struct {
	SSHServer   *SSHServer
	GitDaemon   *GitDaemon
	HTTPServer  *HTTPServer
	StatsServer *StatsServer
	Cron        *cron.CronScheduler
	Config      *config.Config
	Backend     backend.Backend
}

// NewServer returns a new *ssh.Server configured to serve Soft Serve. The SSH
// server key-pair will be created if none exists. An initial admin SSH public
// key can be provided with authKey. If authKey is provided, access will be
// restricted to that key. If authKey is not provided, the server will be
// publicly writable until configured otherwise by cloning the `config` repo.
func NewServer(cfg *config.Config) (*Server, error) {
	var err error
	if cfg.Backend == nil {
		sb, err := sqlite.NewSqliteBackend(cfg.DataPath)
		if err != nil {
			logger.Fatal(err)
		}

		// Add the initial admin keys to the list of admins.
		sb.AdditionalAdmins = cfg.InitialAdminKeys
		cfg = cfg.WithBackend(sb)

		// Create internal key.
		_, err = keygen.NewWithWrite(
			filepath.Join(cfg.DataPath, cfg.SSH.InternalKeyPath),
			nil,
			keygen.Ed25519,
		)
		if err != nil {
			return nil, err
		}
	}

	srv := &Server{
		Cron:    cron.NewCronScheduler(),
		Config:  cfg,
		Backend: cfg.Backend,
	}

	// Add cron jobs.
	srv.Cron.AddFunc(jobSpecs["mirror"], mirrorJob(cfg.Backend))

	srv.SSHServer, err = NewSSHServer(cfg, srv)
	if err != nil {
		return nil, err
	}

	srv.GitDaemon, err = NewGitDaemon(cfg)
	if err != nil {
		return nil, err
	}

	srv.HTTPServer, err = NewHTTPServer(cfg)
	if err != nil {
		return nil, err
	}

	srv.StatsServer, err = NewStatsServer(cfg)
	if err != nil {
		return nil, err
	}

	return srv, nil
}

// Start starts the SSH server.
func (s *Server) Start() error {
	var errg errgroup.Group
	errg.Go(func() error {
		log.Print("Starting Git daemon", "addr", s.Config.Git.ListenAddr)
		if err := s.GitDaemon.Start(); err != ErrServerClosed {
			return err
		}
		return nil
	})
	errg.Go(func() error {
		log.Print("Starting HTTP server", "addr", s.Config.HTTP.ListenAddr)
		if err := s.HTTPServer.ListenAndServe(); err != http.ErrServerClosed {
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
	errg.Go(func() error {
		log.Print("Starting Stats server", "addr", s.Config.Stats.ListenAddr)
		if err := s.StatsServer.ListenAndServe(); err != http.ErrServerClosed {
			return err
		}
		return nil
	})
	errg.Go(func() error {
		log.Print("Starting cron scheduler")
		s.Cron.Start()
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
		return s.HTTPServer.Shutdown(ctx)
	})
	errg.Go(func() error {
		return s.SSHServer.Shutdown(ctx)
	})
	errg.Go(func() error {
		return s.StatsServer.Shutdown(ctx)
	})
	return errg.Wait()
}

// Close closes the SSH server.
func (s *Server) Close() error {
	var errg errgroup.Group
	errg.Go(s.GitDaemon.Close)
	errg.Go(s.HTTPServer.Close)
	errg.Go(s.SSHServer.Close)
	errg.Go(s.StatsServer.Close)
	return errg.Wait()
}
