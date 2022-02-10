package server

import (
	"context"
	"log"

	"github.com/charmbracelet/soft-serve/config"
	appCfg "github.com/charmbracelet/soft-serve/internal/config"
)

// Server is the Soft Serve server.
type Server struct {
	HTTPServer *HTTPServer
	SSHServer  *SSHServer
	Cfg        *config.Config
	ac         *appCfg.Config
}

// NewServer returns a new *ssh.Server configured to serve Soft Serve. The SSH
// server key-pair will be created if none exists. An initial admin SSH public
// key can be provided with authKey. If authKey is provided, access will be
// restricted to that key. If authKey is not provided, the server will be
// publicly writable until configured otherwise by cloning the `config` repo.
func NewServer(cfg *config.Config) *Server {
	ac, err := appCfg.NewConfig(cfg)
	if err != nil {
		log.Fatal(err)
	}
	return &Server{
		HTTPServer: NewHTTPServer(cfg, ac),
		SSHServer:  NewSSHServer(cfg, ac),
		Cfg:        cfg,
		ac:         ac,
	}
}

// Reload reloads the server configuration.
func (srv *Server) Reload() error {
	return srv.ac.Reload()
}

func (s *Server) Start() {
	go func() {
		log.Printf("Starting HTTP server on %s:%d", s.Cfg.BindAddr, s.Cfg.HTTPPort)
		if err := s.HTTPServer.Start(); err != nil {
			log.Fatal(err)
		}
	}()
	log.Printf("Starting SSH server on %s:%d", s.Cfg.BindAddr, s.Cfg.SSHPort)
	if err := s.SSHServer.Start(); err != nil {
		log.Fatal(err)
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	log.Printf("Stopping SSH server on %s:%d", s.Cfg.BindAddr, s.Cfg.SSHPort)
	err := s.SSHServer.Shutdown(ctx)
	if err != nil {
		return err
	}
	log.Printf("Stopping HTTP server on %s:%d", s.Cfg.BindAddr, s.Cfg.SSHPort)
	return s.HTTPServer.Shutdown(ctx)
}
