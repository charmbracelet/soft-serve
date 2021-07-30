package main

import (
	"smoothie/tui"

	"github.com/meowgorithm/babyenv"
)

type Config struct {
	KeyPath string `env:"SMOOTHIE_KEY_PATH" default:".ssh/smoothie_server_ed25519"`
	Port    int    `env:"SMOOTHIE_PORT" default:"23231"`
}

func main() {
	var cfg Config
	err := babyenv.Parse(&cfg)
	if err != nil {
		panic(err)
	}
	s, err := NewServer(cfg.Port, cfg.KeyPath, LoggingMiddleware(), BubbleTeaMiddleware(tui.SessionHandler))
	if err != nil {
		panic(err)
	}
	err = s.Start()
	if err != nil {
		panic(err)
	}
}
