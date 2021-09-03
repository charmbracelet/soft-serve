package main

import (
	"fmt"
	"log"
	"soft-serve/tui"
	"time"

	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	gm "github.com/charmbracelet/wish/git"
	lm "github.com/charmbracelet/wish/logging"

	"github.com/meowgorithm/babyenv"
)

type Config struct {
	Port         int    `env:"SOFT_SERVE_PORT" default:"23231"`
	Host         string `env:"SOFT_SERVE_HOST" default:""`
	KeyPath      string `env:"SOFT_SERVE_KEY_PATH" default:".ssh/soft_serve_server_ed25519"`
	RepoAuth     string `env:"SOFT_SERVE_REPO_KEYS" default:""`
	RepoAuthFile string `env:"SOFT_SERVE_REPO_KEYS_PATH" default:".ssh/soft_serve_git_authorized_keys"`
	RepoPath     string `env:"SOFT_SERVE_REPO_PATH" default:".repos"`
}

func main() {
	var cfg Config
	err := babyenv.Parse(&cfg)
	if err != nil {
		log.Fatalln(err)
	}
	s, err := wish.NewServer(
		wish.WithAddress(fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)),
		wish.WithHostKeyPath(cfg.KeyPath),
		wish.WithMiddlewares(
			bm.Middleware(tui.SessionHandler(cfg.RepoPath, time.Second*5)),
			gm.Middleware(cfg.RepoPath, cfg.RepoAuth, cfg.RepoAuthFile),
			lm.Middleware(),
		),
	)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("Starting SSH server on %s:%d\n", cfg.Host, cfg.Port)
	err = s.ListenAndServe()
	if err != nil {
		log.Fatalln(err)
	}
}
