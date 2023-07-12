package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/charmbracelet/log"

	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/cron"
	"github.com/charmbracelet/soft-serve/server/daemon"
	"github.com/charmbracelet/soft-serve/server/db"
	sshsrv "github.com/charmbracelet/soft-serve/server/ssh"
	"github.com/charmbracelet/soft-serve/server/stats"
	"github.com/charmbracelet/soft-serve/server/web"
	"github.com/charmbracelet/ssh"
	"golang.org/x/sync/errgroup"
)

// Server is the Soft Serve server.
type Server struct {
	SSHServer   *sshsrv.SSHServer
	GitDaemon   *daemon.GitDaemon
	HTTPServer  *web.HTTPServer
	StatsServer *stats.StatsServer
	Cron        *cron.CronScheduler
	Config      *config.Config
	Backend     *backend.Backend
	DB          *db.DB

	logger *log.Logger
	ctx    context.Context
}

// NewServer returns a new *ssh.Server configured to serve Soft Serve. The SSH
// server key-pair will be created if none exists. An initial admin SSH public
// key can be provided with authKey. If authKey is provided, access will be
// restricted to that key. If authKey is not provided, the server will be
// publicly writable until configured otherwise by cloning the `config` repo.
func NewServer(ctx context.Context, db *db.DB) (*Server, error) {
	var err error
	cfg := config.FromContext(ctx)
	be := backend.New(ctx, cfg, db)
	ctx = backend.WithContext(ctx, be)
	srv := &Server{
		Cron:    cron.NewCronScheduler(ctx),
		Config:  cfg,
		Backend: be,
		DB:      db,
		logger:  log.FromContext(ctx).WithPrefix("server"),
		ctx:     ctx,
	}

	// Add cron jobs.
	_, _ = srv.Cron.AddFunc(jobSpecs["mirror"], srv.mirrorJob(be))

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

	return srv, nil
}

// Start starts the SSH server.
func (s *Server) Start() error {
	errg, _ := errgroup.WithContext(s.ctx)
	errg.Go(func() error {
		s.logger.Print("Starting Git daemon", "addr", s.Config.Git.ListenAddr)
		if err := s.GitDaemon.Start(); !errors.Is(err, daemon.ErrServerClosed) {
			return err
		}
		return nil
	})
	errg.Go(func() error {
		s.logger.Print("Starting HTTP server", "addr", s.Config.HTTP.ListenAddr)
		if err := s.HTTPServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})
	errg.Go(func() error {
		s.logger.Print("Starting SSH server", "addr", s.Config.SSH.ListenAddr)
		if err := s.SSHServer.ListenAndServe(); !errors.Is(err, ssh.ErrServerClosed) {
			return err
		}
		return nil
	})
	errg.Go(func() error {
		s.logger.Print("Starting Stats server", "addr", s.Config.Stats.ListenAddr)
		if err := s.StatsServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})
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
