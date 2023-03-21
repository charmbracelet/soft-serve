package config

import (
	glog "log"
	"path/filepath"

	"github.com/caarlos0/env/v6"
	"github.com/charmbracelet/log"
)

// Callbacks provides an interface that can be used to run callbacks on different events.
type Callbacks interface {
	Tui(action string)
	Push(repo string)
	Fetch(repo string)
}

// Config is the configuration for Soft Serve.
type Config struct {
	BindAddr         string   `env:"SOFT_SERVE_BIND_ADDRESS" envDefault:""`
	Host             string   `env:"SOFT_SERVE_HOST" envDefault:"localhost"`
	Port             int      `env:"SOFT_SERVE_PORT" envDefault:"23231"`
	KeyPath          string   `env:"SOFT_SERVE_KEY_PATH"`
	RepoPath         string   `env:"SOFT_SERVE_REPO_PATH" envDefault:".repos"`
	Debug            bool     `env:"SOFT_SERVE_DEBUG" envDefault:"false"`
	InitialAdminKeys []string `env:"SOFT_SERVE_INITIAL_ADMIN_KEY" envSeparator:"\n"`
	Callbacks        Callbacks
	ErrorLog         *glog.Logger
}

// DefaultConfig returns a Config with the values populated with the defaults
// or specified environment variables.
func DefaultConfig() *Config {
	cfg := &Config{ErrorLog: log.StandardLog(log.StandardLogOptions{ForceLevel: log.ErrorLevel})}
	if err := env.Parse(cfg); err != nil {
		log.Fatal(err)
	}
	if cfg.Debug {
		log.SetLevel(log.DebugLevel)
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
func (c *Config) WithErrorLogger(logger *glog.Logger) *Config {
	c.ErrorLog = logger
	return c
}
