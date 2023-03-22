package config

import (
	glog "log"

	"github.com/caarlos0/env/v6"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/backend/file"
)

// SSHConfig is the configuration for the SSH server.
type SSHConfig struct {
	// ListenAddr is the address on which the SSH server will listen.
	ListenAddr string `env:"LISTEN_ADDR" envDefault:":23231"`

	// KeyPath is the path to the SSH server's private key.
	KeyPath string `env:"KEY_PATH" envDefault:"soft_serve"`
}

// GitConfig is the Git daemon configuration for the server.
type GitConfig struct {
	// ListenAddr is the address on which the Git daemon will listen.
	ListenAddr string `env:"LISTEN_ADDR" envDefault:":9418"`

	// MaxTimeout is the maximum number of seconds a connection can take.
	MaxTimeout int `env:"MAX_TIMEOUT" envDefault:"0"`

	// IdleTimeout is the number of seconds a connection can be idle before it is closed.
	IdleTimeout int `env:"IDLE_TIMEOUT" envDefault:"3"`

	// MaxConnections is the maximum number of concurrent connections.
	MaxConnections int `env:"MAX_CONNECTIONS" envDefault:"32"`
}

// Config is the configuration for Soft Serve.
type Config struct {
	// SSH is the configuration for the SSH server.
	SSH SSHConfig `env:"SSH", envPrefix:"SSH_"`

	// Git is the configuration for the Git daemon.
	Git GitConfig `env:"GIT", envPrefix:"GIT_"`

	// InitialAdminKeys is a list of public keys that will be added to the list of admins.
	InitialAdminKeys []string `env:"INITIAL_ADMIN_KEY" envSeparator:"\n"`

	// DataPath is the path to the directory where Soft Serve will store its data.
	DataPath string `env:"DATA_PATH" envDefault:"data"`

	// Deprecated: use DataPath instead.
	KeyPath string `env:"KEY_PATH"`

	// Deprecated: use DataPath instead.
	RepoPath string `env:"REPO_PATH" envDefault:".repos"`

	Debug bool `env:"DEBUG" envDefault:"false"`

	ErrorLog *glog.Logger

	// Backend is the Git backend to use.
	Backend backend.Backend

	// Access is the access control backend to use.
	Access backend.AccessMethod
}

// DefaultConfig returns a Config with the values populated with the defaults
// or specified environment variables.
func DefaultConfig() *Config {
	cfg := &Config{ErrorLog: log.StandardLog(log.StandardLogOptions{ForceLevel: log.ErrorLevel})}
	if err := env.Parse(cfg, env.Options{
		Prefix: "SOFT_SERVE_",
	}); err != nil {
		log.Fatal(err)
	}
	if cfg.Debug {
		log.SetLevel(log.DebugLevel)
	}
	fb, err := file.NewFileBackend(cfg.DataPath)
	if err != nil {
		log.Fatal(err)
	}
	return cfg.WithBackend(fb).WithAccessMethod(fb)
}

// WithErrorLogger sets the error logger for the configuration.
func (c *Config) WithErrorLogger(logger *glog.Logger) *Config {
	c.ErrorLog = logger
	return c
}

// WithBackend sets the backend for the configuration.
func (c *Config) WithBackend(backend backend.Backend) *Config {
	c.Backend = backend
	return c
}

// WithAccessMethod sets the access control method for the configuration.
func (c *Config) WithAccessMethod(access backend.AccessMethod) *Config {
	c.Access = access
	return c
}
