package main

import (
	"fmt"
	"log"
	"smoothie/tui"
	"time"

	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	gm "github.com/charmbracelet/wish/git"
	lm "github.com/charmbracelet/wish/logging"

	"github.com/meowgorithm/babyenv"
)

type Config struct {
	Port         int    `env:"SMOOTHIE_PORT" default:"23231"`
	Host         string `env:"SMOOTHIE_HOST" default:""`
	KeyPath      string `env:"SMOOTHIE_KEY_PATH" default:".ssh/smoothie_server_ed25519"`
	RepoAuth     string `env:"SMOOTHIE_REPO_KEYS" default:""`
	RepoAuthFile string `env:"SMOOTHIE_REPO_KEYS_PATH" default:".ssh/smoothie_git_authorized_keys"`
	RepoPath     string `env:"SMOOTHIE_REPO_PATH" default:".repos"`
}

func main() {
	var cfg Config
	err := babyenv.Parse(&cfg)
	if err != nil {
		log.Fatalln(err)
	}
	s, err := wish.NewServer(
		fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		cfg.KeyPath,
		bm.Middleware(tui.SessionHandler(cfg.RepoPath, time.Second*5)),
		gm.Middleware(cfg.RepoPath, cfg.RepoAuth, cfg.RepoAuthFile),
		lm.Middleware(),
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
