package web

import (
	"context"
	"net/http"
	"time"

	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/config"
)

// HTTPServer is an http server.
type HTTPServer struct {
	ctx    context.Context
	cfg    *config.Config
	be     backend.Backend
	server *http.Server
}

// NewHTTPServer creates a new HTTP server.
func NewHTTPServer(ctx context.Context) (*HTTPServer, error) {
	cfg := config.FromContext(ctx)
	s := &HTTPServer{
		ctx: ctx,
		cfg: cfg,
		be:  backend.FromContext(ctx),
		server: &http.Server{
			Addr:              cfg.HTTP.ListenAddr,
			Handler:           NewRouter(ctx),
			ReadHeaderTimeout: time.Second * 10,
			ReadTimeout:       time.Second * 10,
			WriteTimeout:      time.Second * 10,
			MaxHeaderBytes:    http.DefaultMaxHeaderBytes,
		},
	}

	return s, nil
}

// Close closes the HTTP server.
func (s *HTTPServer) Close() error {
	return s.server.Close()
}

// ListenAndServe starts the HTTP server.
func (s *HTTPServer) ListenAndServe() error {
	if s.cfg.HTTP.TLSKeyPath != "" && s.cfg.HTTP.TLSCertPath != "" {
		return s.server.ListenAndServeTLS(s.cfg.HTTP.TLSCertPath, s.cfg.HTTP.TLSKeyPath)
	}
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the HTTP server.
func (s *HTTPServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
