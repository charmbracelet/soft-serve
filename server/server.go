package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/charmbracelet/log"

	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/backend/sqlite"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/cron"
	"github.com/charmbracelet/soft-serve/server/daemon"
	"github.com/charmbracelet/soft-serve/server/internal"
	sshsrv "github.com/charmbracelet/soft-serve/server/ssh"
	"github.com/charmbracelet/soft-serve/server/stats"
	"github.com/charmbracelet/soft-serve/server/web"
	"github.com/charmbracelet/ssh"
	"golang.org/x/sync/errgroup"
)

var (
	logger = log.WithPrefix("server")
)

// Server is the Soft Serve server.
type Server struct {
	SSHServer      *sshsrv.SSHServer
	GitDaemon      *daemon.GitDaemon
	HTTPServer     *web.HTTPServer
	StatsServer    *stats.StatsServer
	InternalServer *internal.InternalServer
	Cron           *cron.CronScheduler
	Config         *config.Config
	Backend        backend.Backend
	ctx            context.Context
}

// NewServer returns a new *ssh.Server configured to serve Soft Serve. The SSH
// server key-pair will be created if none exists. An initial admin SSH public
// key can be provided with authKey. If authKey is provided, access will be
// restricted to that key. If authKey is not provided, the server will be
// publicly writable until configured otherwise by cloning the `config` repo.
func NewServer(ctx context.Context, cfg *config.Config) (*Server, error) {
	var err error
	if cfg.Backend == nil {
		sb, err := sqlite.NewSqliteBackend(ctx, cfg)
		if err != nil {
			return nil, fmt.Errorf("create backend: %w", err)
		}

		cfg = cfg.WithBackend(sb)
	}

	srv := &Server{
		Cron:    cron.NewCronScheduler(ctx),
		Config:  cfg,
		Backend: cfg.Backend,
		ctx:     ctx,
	}

	// Add cron jobs.
	srv.Cron.AddFunc(jobSpecs["mirror"], mirrorJob(cfg))

	srv.SSHServer, err = sshsrv.NewSSHServer(cfg)
	if err != nil {
		return nil, fmt.Errorf("create ssh server: %w", err)
	}

	srv.GitDaemon, err = daemon.NewGitDaemon(cfg)
	if err != nil {
		return nil, fmt.Errorf("create git daemon: %w", err)
	}

	srv.HTTPServer, err = web.NewHTTPServer(cfg)
	if err != nil {
		return nil, fmt.Errorf("create http server: %w", err)
	}

	srv.StatsServer, err = stats.NewStatsServer(cfg)
	if err != nil {
		return nil, fmt.Errorf("create stats server: %w", err)
	}

	srv.InternalServer, err = internal.NewInternalServer(cfg, srv)
	if err != nil {
		return nil, fmt.Errorf("create internal server: %w", err)
	}

	return srv, nil
}

func start(ctx context.Context, fn func() error) error {
	errc := make(chan error, 1)
	go func() {
		errc <- fn()
	}()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Start starts the SSH server.
func (s *Server) Start() error {
	logger := log.FromContext(s.ctx).WithPrefix("server")
	errg, ctx := errgroup.WithContext(s.ctx)
	errg.Go(func() error {
		logger.Print("Starting Git daemon", "addr", s.Config.Git.ListenAddr)
		if err := start(ctx, s.GitDaemon.Start); !errors.Is(err, daemon.ErrServerClosed) {
			return err
		}
		return nil
	})
	errg.Go(func() error {
		logger.Print("Starting HTTP server", "addr", s.Config.HTTP.ListenAddr)
		if err := start(ctx, s.HTTPServer.ListenAndServe); !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})
	errg.Go(func() error {
		logger.Print("Starting SSH server", "addr", s.Config.SSH.ListenAddr)
		if err := start(ctx, s.SSHServer.ListenAndServe); !errors.Is(err, ssh.ErrServerClosed) {
			return err
		}
		return nil
	})
	errg.Go(func() error {
		logger.Print("Starting Stats server", "addr", s.Config.Stats.ListenAddr)
		if err := start(ctx, s.StatsServer.ListenAndServe); !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})
	errg.Go(func() error {
		logger.Print("Starting cron scheduler")
		s.Cron.Start()
		return nil
	})
	errg.Go(func() error {
		logger.Print("Starting internal server", "addr", s.Config.Internal.ListenAddr)
		if err := start(ctx, s.InternalServer.Start); !errors.Is(err, http.ErrServerClosed) {
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
	errg.Go(func() error {
		return s.InternalServer.Shutdown(ctx)
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
	errg.Go(func() error {
		s.Cron.Stop()
		return nil
	})
	errg.Go(s.InternalServer.Close)
	return errg.Wait()
}
