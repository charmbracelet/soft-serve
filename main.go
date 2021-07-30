package main

import (
	"log"
	"smoothie/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/meowgorithm/babyenv"
)

type Config struct {
	Port     int    `env:"SMOOTHIE_PORT" default:"23231"`
	KeyPath  string `env:"SMOOTHIE_KEY_PATH" default:".ssh/smoothie_server_ed25519"`
	RepoPath string `env:"SMOOTHIE_REPO_PATH" default:".repos"`
}

func main() {
	var cfg Config
	err := babyenv.Parse(&cfg)
	if err != nil {
		log.Fatalln(err)
	}
	btm := BubbleTeaMiddleware(tui.SessionHandler, tea.WithAltScreen())
	gm := GitMiddleware(cfg.RepoPath)
	s, err := NewServer(cfg.Port, cfg.KeyPath, btm, gm, LoggingMiddleware())
	if err != nil {
		log.Fatalln(err)
	}
	err = s.Start()
	if err != nil {
		log.Fatalln(err)
	}
}
