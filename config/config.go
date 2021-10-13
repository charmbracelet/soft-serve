package config

import (
	"log"

	"github.com/charmbracelet/soft/stats"
	"github.com/meowgorithm/babyenv"
)

// Config is the configuration for the soft-serve.
type Config struct {
	Host            string `env:"SOFT_SERVE_HOST" default:""`
	Port            int    `env:"SOFT_SERVE_PORT" default:"23231"`
	KeyPath         string `env:"SOFT_SERVE_KEY_PATH" default:".ssh/soft_serve_server_ed25519"`
	RepoPath        string `env:"SOFT_SERVE_REPO_PATH" default:".repos"`
	InitialAdminKey string `env:"SOFT_SERVE_INITIAL_ADMIN_KEY" default:""`
	Stats           stats.Stats
}

// DefaultConfig returns a Config with the values populated with the defaults
// or specified environment variables.
func DefaultConfig() *Config {
	var scfg Config
	err := babyenv.Parse(&scfg)
	if err != nil {
		log.Fatalln(err)
	}
	return scfg.WithStats(stats.NewStats())
}

func (cfg *Config) WithStats(s stats.Stats) *Config {
	cfg.Stats = s
	return cfg
}
