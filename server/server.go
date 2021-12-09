package server

import (
	"fmt"
	"log"

	"github.com/charmbracelet/soft-serve/config"
	appCfg "github.com/charmbracelet/soft-serve/internal/config"
	"github.com/charmbracelet/soft-serve/internal/tui"

	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	gm "github.com/charmbracelet/wish/git"
	lm "github.com/charmbracelet/wish/logging"
	"github.com/gliderlabs/ssh"
)

// Server is the Soft Serve server.
type Server struct {
	SSHServer *ssh.Server
	Config    *config.Config
	config    *appCfg.Config
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
	mw := []wish.Middleware{
		bm.Middleware(tui.SessionHandler(ac)),
		gm.Middleware(cfg.RepoPath, ac),
		lm.Middleware(),
	}
	s, err := wish.NewServer(
		ssh.PublicKeyAuth(ac.PublicKeyHandler),
		ssh.PasswordAuth(ac.PasswordHandler),
		wish.WithAddress(fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)),
		wish.WithHostKeyPath(cfg.KeyPath),
		wish.WithMiddleware(mw...),
	)
	if err != nil {
		log.Fatalln(err)
	}
	return &Server{
		SSHServer: s,
		Config:    cfg,
		config:    ac,
	}
}

// Reload reloads the server configuration.
func (srv *Server) Reload() error {
	return srv.config.Reload()
}

// Start starts the SSH server.
func (srv *Server) Start() error {
	return srv.SSHServer.ListenAndServe()
}
