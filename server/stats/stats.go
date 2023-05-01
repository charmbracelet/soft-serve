package stats

import (
	"context"
	"net/http"
	"time"

	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// StatsServer is a server for collecting and reporting statistics.
type StatsServer struct {
	ctx    context.Context
	cfg    *config.Config
	server *http.Server
}

// NewStatsServer returns a new StatsServer.
func NewStatsServer(ctx context.Context) (*StatsServer, error) {
	cfg := config.FromContext(ctx)
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	return &StatsServer{
		ctx: ctx,
		cfg: cfg,
		server: &http.Server{
			Addr:              cfg.Stats.ListenAddr,
			Handler:           mux,
			ReadHeaderTimeout: time.Second * 10,
			ReadTimeout:       time.Second * 10,
			WriteTimeout:      time.Second * 10,
			MaxHeaderBytes:    http.DefaultMaxHeaderBytes,
		},
	}, nil
}

// ListenAndServe starts the StatsServer.
func (s *StatsServer) ListenAndServe() error {
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the StatsServer.
func (s *StatsServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// Close closes the StatsServer.
func (s *StatsServer) Close() error {
	return s.server.Close()
}
