package serve

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"

	"charm.land/log/v2"

	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/cron"
	"github.com/charmbracelet/soft-serve/pkg/daemon"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/jobs"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	sshsrv "github.com/charmbracelet/soft-serve/pkg/ssh"
	"github.com/charmbracelet/soft-serve/pkg/stats"
	"github.com/charmbracelet/soft-serve/pkg/utils"
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
		jobCfg, err := j.Runner.Config(ctx)
		if err != nil {
			return nil, fmt.Errorf("parse cronjob [%s] config: %w", n, err)
		}
		spec := jobCfg.Spec()
		if spec == "" {
			continue
		}

		id, err := sched.AddFunc(spec, j.Runner.Func(ctx, jobCfg))
		if err != nil {
			logger.Warn("error adding cron job", "job", n, "err", err)
		}

		j.ID = id
	}

	srv.Cron = sched

	ensureDefaultRepo(ctx, cfg, be, logger)

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

	warnIfAnonAdminAccess(ctx, be, logger)

	return srv, nil
}

// warnIfAnonAdminAccess logs a loud warning if the server's effective,
// post-override settings grant unauthenticated (keyless) connections admin
// access. This checks effective runtime state via the backend, not just the
// new config-override fields, since the same risk exists whether the
// dangerous combination came from config or was set via `ssh soft settings`
// on a previous run.
func warnIfAnonAdminAccess(ctx context.Context, be *backend.Backend, logger *log.Logger) {
	if !be.AllowKeyless(ctx) || be.AnonAccess(ctx) < access.AdminAccess {
		return
	}

	logger.Warn("################################################################")
	logger.Warn("# WARNING: anonymous keyless connections have ADMIN access.    #")
	logger.Warn("# Anyone who can reach this server has full control, no auth.  #")
	logger.Warn("# This is intended for local/dev use only. Do not expose this  #")
	logger.Warn("# server to an untrusted network.                              #")
	logger.Warn("################################################################")
}

// ensureDefaultRepo creates the repo named by cfg.DefaultRepo if it does not
// already exist. It never fails startup: it logs invalid names and creation
// errors, then returns.
func ensureDefaultRepo(ctx context.Context, cfg *config.Config, be *backend.Backend, logger *log.Logger) {
	if cfg.DefaultRepo == "" {
		return
	}

	name := utils.SanitizeRepo(cfg.DefaultRepo)
	if err := utils.ValidateRepo(name); err != nil {
		logger.Warn("invalid default_repo, skipping", "name", cfg.DefaultRepo, "err", err)
		return
	}

	if _, err := be.Repository(ctx, name); err == nil {
		return
	} else if !errors.Is(err, proto.ErrRepoNotFound) {
		logger.Warn("failed to look up default repo", "name", name, "err", err)
		return
	}

	// The migration always inserts a user at ID 1. The repos table requires
	// a non-null owner, so we attribute the repo to that account.
	owner, err := be.UserByID(ctx, 1)
	if err != nil {
		logger.Warn("failed to look up default repo owner, skipping", "name", name, "err", err)
		return
	}

	if _, err := be.CreateRepository(ctx, name, owner, proto.RepositoryOptions{}); err != nil && !errors.Is(err, proto.ErrRepoExist) {
		logger.Warn("failed to create default repo", "name", name, "err", err)
		return
	}

	logger.Info("created default repo", "name", name)
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
	errg, _ := errgroup.WithContext(s.ctx)

	// optionally start the SSH server
	if s.Config.SSH.Enabled {
		errg.Go(func() error {
			s.logger.Print("Starting SSH server", "addr", s.Config.SSH.ListenAddr)
			if err := s.SSHServer.ListenAndServe(); !errors.Is(err, ssh.ErrServerClosed) {
				return err
			}
			return nil
		})
	}

	// optionally start the git daemon
	if s.Config.Git.Enabled {
		errg.Go(func() error {
			s.logger.Print("Starting Git daemon", "addr", s.Config.Git.ListenAddr)
			if err := s.GitDaemon.ListenAndServe(); !errors.Is(err, daemon.ErrServerClosed) {
				return err
			}
			return nil
		})
	}

	// optionally start the HTTP server
	if s.Config.HTTP.Enabled {
		errg.Go(func() error {
			s.logger.Print("Starting HTTP server", "addr", s.Config.HTTP.ListenAddr)
			if err := s.HTTPServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
				return err
			}
			return nil
		})
	}

	// optionally start the Stats server
	if s.Config.Stats.Enabled {
		errg.Go(func() error {
			s.logger.Print("Starting Stats server", "addr", s.Config.Stats.ListenAddr)
			if err := s.StatsServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
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
			// jobID from github.com/robfig/cron/v2 starts from 1
			if j.ID != 0 {
				s.Cron.Remove(j.ID)
			}
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
