package web

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"time"

	"charm.land/log/v2"
	"github.com/charmbracelet/soft-serve/pkg/config"
)

// HTTPServer is an http server.
type HTTPServer struct {
	ctx context.Context
	cfg *config.Config

	Server *http.Server
}

// NewHTTPServer creates a new HTTP server.
func NewHTTPServer(ctx context.Context) (*HTTPServer, error) {
	cfg := config.FromContext(ctx)
	logger := log.FromContext(ctx)
	s := &HTTPServer{
		ctx: ctx,
		cfg: cfg,
		Server: &http.Server{
			Addr:              cfg.HTTP.ListenAddr,
			Handler:           NewRouter(ctx),
			ReadHeaderTimeout: time.Second * 10,
			WriteTimeout:      5 * time.Minute,
			IdleTimeout:       time.Second * 10,
			MaxHeaderBytes:    http.DefaultMaxHeaderBytes,
			ErrorLog:          logger.StandardLog(log.StandardLogOptions{ForceLevel: log.ErrorLevel}),
		},
	}

	return s, nil
}

// SetTLSConfig sets the TLS configuration for the HTTP server.
func (s *HTTPServer) SetTLSConfig(tlsConfig *tls.Config) {
	s.Server.TLSConfig = tlsConfig
}

// Close closes the HTTP server.
func (s *HTTPServer) Close() error {
	return s.Server.Close()
}

// Serve accepts connections on l and serves HTTP requests.
func (s *HTTPServer) Serve(l net.Listener) error {
	if s.Server.TLSConfig != nil {
		// ServeTLS with empty cert/key paths is only valid when at least
		// one certificate source is set on the TLSConfig: Certificates,
		// GetCertificate, or GetConfigForClient (which can supply a full
		// tls.Config dynamically, e.g. for SNI-based routing).
		tlsCfg := s.Server.TLSConfig
		if len(tlsCfg.Certificates) == 0 &&
			tlsCfg.GetCertificate == nil &&
			tlsCfg.GetConfigForClient == nil {
			return errors.New("TLS configured but no certificate source provided (set Certificates, GetCertificate, or GetConfigForClient)")
		}
		return s.Server.ServeTLS(l, "", "")
	}
	return s.Server.Serve(l)
}

// ListenAndServe starts the HTTP server.
func (s *HTTPServer) ListenAndServe() error {
	if s.Server.TLSConfig != nil {
		tlsCfg := s.Server.TLSConfig
		if len(tlsCfg.Certificates) == 0 &&
			tlsCfg.GetCertificate == nil &&
			tlsCfg.GetConfigForClient == nil {
			return errors.New("TLS configured but no certificate source provided (set Certificates, GetCertificate, or GetConfigForClient)")
		}
		return s.Server.ListenAndServeTLS("", "")
	}
	return s.Server.ListenAndServe()
}

// Shutdown gracefully shuts down the HTTP server.
func (s *HTTPServer) Shutdown(ctx context.Context) error {
	return s.Server.Shutdown(ctx)
}
