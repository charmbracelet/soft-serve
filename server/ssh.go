package server

import (
	"context"
	"fmt"
	"log"

	"github.com/charmbracelet/soft-serve/config"
	appCfg "github.com/charmbracelet/soft-serve/internal/config"
	"github.com/charmbracelet/soft-serve/internal/tui"
	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	gm "github.com/charmbracelet/wish/git"
	lm "github.com/charmbracelet/wish/logging"
	rm "github.com/charmbracelet/wish/recover"
	"github.com/gliderlabs/ssh"
)

type SSHServer struct {
	server *ssh.Server
	cfg    *config.Config
	ac     *appCfg.Config
}

// NewServer returns a new *ssh.Server configured to serve Soft Serve. The SSH
// server key-pair will be created if none exists. An initial admin SSH public
// key can be provided with authKey. If authKey is provided, access will be
// restricted to that key. If authKey is not provided, the server will be
// publicly writable until configured otherwise by cloning the `config` repo.
func NewSSHServer(cfg *config.Config, ac *appCfg.Config) *SSHServer {
	mw := []wish.Middleware{
		rm.MiddlewareWithLogger(
			cfg.ErrorLog,
			softServeMiddleware(ac),
			bm.Middleware(tui.SessionHandler(ac)),
			gm.Middleware(cfg.RepoPath, ac),
			lm.Middleware(),
		),
	}
	s, err := wish.NewServer(
		ssh.PublicKeyAuth(ac.PublicKeyHandler),
		ssh.PasswordAuth(ac.PasswordHandler),
		wish.WithAddress(fmt.Sprintf("%s:%d", cfg.BindAddr, cfg.SSHPort)),
		wish.WithHostKeyPath(cfg.KeyPath),
		wish.WithMiddleware(mw...),
	)
	if err != nil {
		log.Fatalln(err)
	}
	return &SSHServer{
		server: s,
		cfg:    cfg,
		ac:     ac,
	}
}

// Start starts the SSH server.
func (s *SSHServer) Start() error {
	return s.server.ListenAndServe()
}

// Shutdown lets the server gracefully shutdown.
func (s *SSHServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
