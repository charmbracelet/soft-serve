package config

import (
	"log"
	"path/filepath"

	"github.com/caarlos0/env/v6"
)

// Callbacks provides an interface that can be used to run callbacks on different events.
type Callbacks interface {
	Tui(action string)
	Push(repo string)
	Fetch(repo string)
}

// Config is the configuration for Soft Serve.
type Config struct {
	Host             string   `env:"SOFT_SERVE_HOST"`
	Port             int      `env:"SOFT_SERVE_PORT"`
	KeyPath          string   `env:"SOFT_SERVE_KEY_PATH"`
	RepoPath         string   `env:"SOFT_SERVE_REPO_PATH"`
	InitialAdminKeys []string `env:"SOFT_SERVE_INITIAL_ADMIN_KEY" envSeparator:"\n"`
	Callbacks        Callbacks
}

func (c *Config) applyDefaults() {
	if c.Port == 0 {
		c.Port = 23231
	}
	if c.KeyPath == "" {
		// NB: cross-platform-compatible path
		c.KeyPath = filepath.Join(".ssh", "soft_serve_server_ed25519")
	}
	if c.RepoPath == "" {
		c.RepoPath = ".repos"
	}
}

// DefaultConfig returns a Config with the values populated with the defaults
// or specified environment variables.
func DefaultConfig() *Config {
	var scfg Config
	if err := env.Parse(&scfg); err != nil {
		log.Fatalln(err)
	}
	scfg.applyDefaults()
	return scfg.WithCallbacks(nil)
}

// WithCallbacks applies the given Callbacks to the configuration.
func (c *Config) WithCallbacks(callbacks Callbacks) *Config {
	c.Callbacks = callbacks
	return c
}
