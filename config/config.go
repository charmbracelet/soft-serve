package config

import (
	"log"

	"github.com/meowgorithm/babyenv"
)

// Callbacks provides an interface that can be used to run callbacks on different events.
type Callbacks interface {
	Tui(action string)
	Push(repo string)
	Fetch(repo string)
}

// Config is the configuration for Soft Serve.
type Config struct {
	Host            string `env:"SOFT_SERVE_HOST" default:""`
	Port            int    `env:"SOFT_SERVE_PORT" default:"23231"`
	KeyPath         string `env:"SOFT_SERVE_KEY_PATH" default:".ssh/soft_serve_server_ed25519"`
	RepoPath        string `env:"SOFT_SERVE_REPO_PATH" default:".repos"`
	InitialAdminKey string `env:"SOFT_SERVE_INITIAL_ADMIN_KEY" default:""`
	Callbacks       Callbacks
}

// DefaultConfig returns a Config with the values populated with the defaults
// or specified environment variables.
func DefaultConfig() *Config {
	var scfg Config
	err := babyenv.Parse(&scfg)
	if err != nil {
		log.Fatalln(err)
	}
	return scfg.WithCallbacks(nil)
}

// WithCallbacks applies the given Callbacks to the configuration.
func (cfg *Config) WithCallbacks(c Callbacks) *Config {
	cfg.Callbacks = c
	return cfg
}
