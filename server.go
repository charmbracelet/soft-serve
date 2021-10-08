package soft

import (
	"fmt"
	"log"

	"github.com/charmbracelet/soft/internal/config"
	"github.com/charmbracelet/soft/internal/git"
	"github.com/charmbracelet/soft/internal/tui"
	"github.com/charmbracelet/soft/stats"
	"github.com/meowgorithm/babyenv"

	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	gm "github.com/charmbracelet/wish/git"
	lm "github.com/charmbracelet/wish/logging"
	"github.com/gliderlabs/ssh"
)

// Config is the configuration for the soft-serve.
type Config struct {
	Host            string `env:"SOFT_SERVE_HOST" default:""`
	Port            int    `env:"SOFT_SERVE_PORT" default:"23231"`
	KeyPath         string `env:"SOFT_SERVE_KEY_PATH" default:".ssh/soft_serve_server_ed25519"`
	RepoPath        string `env:"SOFT_SERVE_REPO_PATH" default:".repos"`
	InitialAdminKey string `env:"SOFT_SERVE_INITIAL_ADMIN_KEY" default:""`
	Stats           stats.Stats
	cfg             *config.Config
}

// DefaultConfig returns a Config with the values populated with the defaults
// or specified environment variables.
func DefaultConfig() *Config {
	var scfg Config
	err := babyenv.Parse(&scfg)
	if err != nil {
		log.Fatalln(err)
	}
	return &scfg
}

// NewServer returns a new *ssh.Server configured to serve Soft Serve. The SSH
// server key-pair will be created if none exists. An initial admin SSH public
// key can be provided with authKey. If authKey is provided, access will be
// restricted to that key. If authKey is not provided, the server will be
// publicly writable until configured otherwise by cloning the `config` repo.
func NewServer(scfg *Config) *ssh.Server {
	rs := git.NewRepoSource(scfg.RepoPath)
	cfg, err := config.NewConfig(scfg.Host, scfg.Port, scfg.InitialAdminKey, rs)
	if err != nil {
		log.Fatalln(err)
	}
	if scfg.Stats != nil {
		cfg = cfg.WithStats(scfg.Stats)
	}
	scfg.cfg = cfg
	mw := []wish.Middleware{
		bm.Middleware(tui.SessionHandler(cfg)),
		gm.Middleware(scfg.RepoPath, cfg),
		lm.Middleware(),
	}
	s, err := wish.NewServer(
		ssh.PublicKeyAuth(cfg.PublicKeyHandler),
		ssh.PasswordAuth(cfg.PasswordHandler),
		wish.WithAddress(fmt.Sprintf("%s:%d", scfg.Host, scfg.Port)),
		wish.WithHostKeyPath(scfg.KeyPath),
		wish.WithMiddleware(mw...),
	)
	if err != nil {
		log.Fatalln(err)
	}
	return s
}
