package main

import (
	"fmt"
	"log"
	"soft-serve/config"
	"soft-serve/git"
	"soft-serve/tui"

	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	gm "github.com/charmbracelet/wish/git"
	lm "github.com/charmbracelet/wish/logging"
	"github.com/gliderlabs/ssh"

	"github.com/meowgorithm/babyenv"
)

type serverConfig struct {
	Port     int    `env:"SOFT_SERVE_PORT" default:"23231"`
	Host     string `env:"SOFT_SERVE_HOST" default:""`
	InitKey  string `env:"SOFT_SERVE_REPO_KEY" default:""`
	KeyPath  string `env:"SOFT_SERVE_KEY_PATH" default:".ssh/soft_serve_server_ed25519"`
	RepoPath string `env:"SOFT_SERVE_REPO_PATH" default:".repos"`
}

func main() {
	var scfg serverConfig
	var cfg *config.Config
	var err error
	err = babyenv.Parse(&scfg)
	if err != nil {
		log.Fatalln(err)
	}
	rs := git.NewRepoSource(scfg.RepoPath)
	cfg, err = config.NewConfig(scfg.Host, scfg.Port, scfg.InitKey, rs)
	if err != nil {
		log.Fatalln(err)
	}
	s, err := wish.NewServer(
		ssh.PublicKeyAuth(cfg.PublicKeyHandler),
		ssh.PasswordAuth(cfg.PasswordHandler),
		wish.WithAddress(fmt.Sprintf("%s:%d", scfg.Host, scfg.Port)),
		wish.WithHostKeyPath(scfg.KeyPath),
		wish.WithMiddlewares(
			bm.Middleware(tui.SessionHandler(cfg)),
			gm.MiddlewareWithPushCallback(scfg.RepoPath, cfg, cfg.Pushed),
			lm.Middleware(),
		),
	)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("Starting SSH server on %s:%d\n", scfg.Host, scfg.Port)
	err = s.ListenAndServe()
	if err != nil {
		log.Fatalln(err)
	}
}
