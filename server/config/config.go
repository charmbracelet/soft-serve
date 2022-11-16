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
	BindAddr      string `env:"SOFT_SERVE_BIND_ADDRESS" envDefault:""`
	Host          string `env:"SOFT_SERVE_HOST" envDefault:"localhost"`
	Port          int    `env:"SOFT_SERVE_PORT" envDefault:"23231"`
	GitPort       int    `env:"SOFT_SERVE_GIT_PORT" envDefault:"9418"`
	GitMaxTimeout int    `env:"SOFT_SERVE_GIT_MAX_TIMEOUT" envDefault:"300"`
	// MaxReadTimeout is the maximum time a client can take to send a request.
	GitMaxReadTimeout int      `env:"SOFT_SERVE_GIT_MAX_READ_TIMEOUT" envDefault:"3"`
	GitMaxConnections int      `env:"SOFT_SERVE_GIT_MAX_CONNECTIONS" envDefault:"32"`
	KeyPath           string   `env:"SOFT_SERVE_KEY_PATH"`
	RepoPath          string   `env:"SOFT_SERVE_REPO_PATH" envDefault:".repos"`
	InitialAdminKeys  []string `env:"SOFT_SERVE_INITIAL_ADMIN_KEY" envSeparator:"\n"`
	Callbacks         Callbacks
	ErrorLog          *log.Logger
}

// DefaultConfig returns a Config with the values populated with the defaults
// or specified environment variables.
func DefaultConfig() *Config {
	cfg := &Config{ErrorLog: log.Default()}
	if err := env.Parse(cfg); err != nil {
		log.Fatalln(err)
	}
	if cfg.KeyPath == "" {
		// NB: cross-platform-compatible path
		cfg.KeyPath = filepath.Join(".ssh", "soft_serve_server_ed25519")
	}
	return cfg.WithCallbacks(nil)
}

// WithCallbacks applies the given Callbacks to the configuration.
func (c *Config) WithCallbacks(callbacks Callbacks) *Config {
	c.Callbacks = callbacks
	return c
}

// WithErrorLogger sets the error logger for the configuration.
func (c *Config) WithErrorLogger(logger *log.Logger) *Config {
	c.ErrorLog = logger
	return c
}
