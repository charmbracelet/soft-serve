package main

import (
	"log"

	"github.com/charmbracelet/soft"

	"github.com/meowgorithm/babyenv"
)

type serverConfig struct {
	Host     string `env:"SOFT_SERVE_HOST" default:""`
	Port     int    `env:"SOFT_SERVE_PORT" default:"23231"`
	KeyPath  string `env:"SOFT_SERVE_KEY_PATH" default:".ssh/soft_serve_server_ed25519"`
	RepoPath string `env:"SOFT_SERVE_REPO_PATH" default:".repos"`
	AuthKey  string `env:"SOFT_SERVE_AUTH_KEY" default:""`
}

func main() {
	var cfg serverConfig
	err := babyenv.Parse(&cfg)
	if err != nil {
		log.Fatalln(err)
	}
	s := soft.NewServer(
		cfg.Host,
		cfg.Port,
		cfg.KeyPath,
		cfg.RepoPath,
		cfg.AuthKey,
	)
	log.Printf("Starting SSH server on %s:%d\n", cfg.Host, cfg.Port)
	err = s.ListenAndServe()
	if err != nil {
		log.Fatalln(err)
	}
}
