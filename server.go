package soft

import (
	"fmt"
	"log"

	"github.com/charmbracelet/soft/config"
	"github.com/charmbracelet/soft/git"
	"github.com/charmbracelet/soft/tui"

	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	gm "github.com/charmbracelet/wish/git"
	lm "github.com/charmbracelet/wish/logging"
	"github.com/gliderlabs/ssh"
)

// NewServer returns a new *ssh.Server configured to serve Soft Serve. The SSH
// server key-pair will be created if none exists. An initial admin SSH public
// key can be provided with authKey. If authKey is provided, access will be
// restricted to that key. If authKey is not provided, the server will be
// publicly writable until configured otherwise by cloning the `config` repo.
func NewServer(host string, port int, serverKeyPath string, repoPath string, authKey string) *ssh.Server {
	rs := git.NewRepoSource(repoPath)
	cfg, err := config.NewConfig(host, port, authKey, rs)
	if err != nil {
		log.Fatalln(err)
	}
	s, err := wish.NewServer(
		ssh.PublicKeyAuth(cfg.PublicKeyHandler),
		ssh.PasswordAuth(cfg.PasswordHandler),
		wish.WithAddress(fmt.Sprintf("%s:%d", host, port)),
		wish.WithHostKeyPath(serverKeyPath),
		wish.WithMiddleware(
			bm.Middleware(tui.SessionHandler(cfg)),
			gm.Middleware(repoPath, cfg),
			lm.Middleware(),
		),
	)
	if err != nil {
		log.Fatalln(err)
	}
	return s
}
