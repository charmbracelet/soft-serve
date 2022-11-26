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

// SSHConfig is the SSH configuration for the server.
type SSHConfig struct {
	Port int `env:"PORT" envDefault:"23231"`
}

// GitConfig is the Git protocol configuration for the server.
type GitConfig struct {
	Port       int `env:"PORT" envDefault:"9418"`
	MaxTimeout int `env:"MAX_TIMEOUT" envDefault:"300"`
	// MaxReadTimeout is the maximum time a client can take to send a request.
	MaxReadTimeout int `env:"MAX_READ_TIMEOUT" envDefault:"3"`
	MaxConnections int `env:"SOFT_SERVE_GIT_MAX_CONNECTIONS" envDefault:"32"`
}

// Config is the configuration for Soft Serve.
type Config struct {
	Host string    `env:"HOST" envDefault:"localhost"`
	SSH  SSHConfig `env:"SSH" envPrefix:"SSH_"`
	Git  GitConfig `env:"GIT" envPrefix:"GIT_"`

	DataPath string `env:"DATA_PATH" envDefault:"soft-serve"`

	// Deprecated: use SOFT_SERVE_SSH_PORT instead.
	Port int `env:"PORT"`
	// Deprecated: use DataPath instead.
	KeyPath string `env:"KEY_PATH"`
	// Deprecated: use DataPath instead.
	ReposPath string `env:"REPO_PATH"`

	InitialAdminKeys []string `env:"INITIAL_ADMIN_KEY" envSeparator:"\n"`
	Callbacks        Callbacks
	ErrorLog         *log.Logger
}

// RepoPath returns the path to the repositories.
func (c *Config) RepoPath() string {
	if c.ReposPath != "" {
		log.Printf("warning: SOFT_SERVE_REPO_PATH is deprecated, use SOFT_SERVE_DATA_PATH instead")
		return c.ReposPath
	}
	return filepath.Join(c.DataPath, "repos")
}

// SSHPath returns the path to the SSH directory.
func (c *Config) SSHPath() string {
	return filepath.Join(c.DataPath, "ssh")
}

// PrivateKeyPath returns the path to the SSH key.
func (c *Config) PrivateKeyPath() string {
	if c.KeyPath != "" {
		log.Printf("warning: SOFT_SERVE_KEY_PATH is deprecated, use SOFT_SERVE_DATA_PATH instead")
		return c.KeyPath
	}
	return filepath.Join(c.DataPath, "ssh", "soft_serve")
}

// DefaultConfig returns a Config with the values populated with the defaults
// or specified environment variables.
func DefaultConfig() *Config {
	cfg := &Config{ErrorLog: log.Default()}
	if err := env.Parse(cfg, env.Options{
		Prefix: "SOFT_SERVE_",
	}); err != nil {
		log.Fatalln(err)
	}
	if cfg.Port != 0 {
		log.Printf("warning: SOFT_SERVE_PORT is deprecated, use SOFT_SERVE_SSH_PORT instead")
		cfg.SSH.Port = cfg.Port
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
