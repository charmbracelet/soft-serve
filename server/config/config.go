package config

import (
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"path/filepath"

	"github.com/caarlos0/env/v6"
	"github.com/charmbracelet/soft-serve/proto"
	"github.com/charmbracelet/soft-serve/server/db"
	"github.com/charmbracelet/soft-serve/server/db/sqlite"
)

// Callbacks provides an interface that can be used to run callbacks on different events.
type Callbacks interface {
	Tui(action string)
	Push(repo string)
	Fetch(repo string)
}

// SSHConfig is the SSH configuration for the server.
type SSHConfig struct {
	Port          int    `env:"PORT" envDefault:"23231"`
	AllowKeyless  bool   `env:"ALLOW_KEYLESS" envDefault:"true"`
	AllowPassword bool   `env:"ALLOW_PASSWORD" envDefault:"false"`
	Password      string `env:"PASSWORD"`
	MaxTimeout    int    `env:"MAX_TIMEOUT" envDefault:"0"`
	IdleTimeout   int    `env:"IDLE_TIMEOUT" envDefault:"300"`
}

// GitConfig is the Git protocol configuration for the server.
type GitConfig struct {
	Port           int `env:"PORT" envDefault:"9418"`
	MaxTimeout     int `env:"MAX_TIMEOUT" envDefault:"0"`
	IdleTimeout    int `env:"IDLE_TIMEOUT" envDefault:"3"`
	MaxConnections int `env:"SOFT_SERVE_GIT_MAX_CONNECTIONS" envDefault:"32"`
}

// DBConfig is the database configuration for the server.
type DBConfig struct {
	Driver   string `env:"DRIVER" envDefault:"sqlite"`
	User     string `env:"USER"`
	Password string `env:"PASSWORD"`
	Host     string `env:"HOST"`
	Port     string `env:"PORT"`
	Name     string `env:"NAME"`
	SSLMode  bool   `env:"SSL_MODE" envDefault:"false"`
}

// URL returns a database URL for the configuration.
func (d *DBConfig) URL() *url.URL {
	switch d.Driver {
	case "sqlite":
		return &url.URL{
			Scheme: "sqlite",
			Path:   filepath.Join(d.Name),
		}
	default:
		ssl := "disable"
		if d.SSLMode {
			ssl = "require"
		}
		var user *url.Userinfo
		if d.User != "" && d.Password != "" {
			user = url.UserPassword(d.User, d.Password)
		} else if d.User != "" {
			user = url.User(d.User)
		}
		return &url.URL{
			Scheme:   d.Driver,
			Host:     net.JoinHostPort(d.Host, d.Port),
			User:     user,
			Path:     d.Name,
			RawQuery: fmt.Sprintf("sslmode=%s", ssl),
		}
	}
}

// Config is the configuration for Soft Serve.
type Config struct {
	Host string    `env:"HOST" envDefault:"localhost"`
	SSH  SSHConfig `env:"SSH" envPrefix:"SSH_"`
	Git  GitConfig `env:"GIT" envPrefix:"GIT_"`
	Db   DBConfig  `env:"DB" envPrefix:"DB_"`

	AnonAccess proto.AccessLevel `env:"ANON_ACCESS" envDefault:"read-only"`
	DataPath   string            `env:"DATA_PATH" envDefault:"data"`

	// Deprecated: use SOFT_SERVE_SSH_PORT instead.
	Port int `env:"PORT"`
	// Deprecated: use DataPath instead.
	KeyPath string `env:"KEY_PATH"`
	// Deprecated: use DataPath instead.
	ReposPath string `env:"REPO_PATH"`

	InitialAdminKeys []string `env:"INITIAL_ADMIN_KEY" envSeparator:"\n"`
	Callbacks        Callbacks
	ErrorLog         *log.Logger

	db db.Store
}

// RepoPath returns the path to the repositories.
func (c *Config) RepoPath() string {
	return filepath.Join(c.DataPath, "repos")
}

// SSHPath returns the path to the SSH directory.
func (c *Config) SSHPath() string {
	return filepath.Join(c.DataPath, "ssh")
}

// PrivateKeyPath returns the path to the SSH key.
func (c *Config) PrivateKeyPath() string {
	return filepath.Join(c.SSHPath(), "soft_serve")
}

// DefaultConfig returns a Config with the values populated with the defaults
// or specified environment variables.
func DefaultConfig() *Config {
	var err error
	var migrateWarn bool
	cfg := &Config{ErrorLog: log.Default()}
	if err = env.Parse(cfg, env.Options{
		Prefix: "SOFT_SERVE_",
	}); err != nil {
		log.Fatalln(err)
	}
	if cfg.Port != 0 {
		log.Printf("warning: SOFT_SERVE_PORT is deprecated, use SOFT_SERVE_SSH_PORT instead.")
		migrateWarn = true
	}
	if cfg.KeyPath != "" {
		log.Printf("warning: SOFT_SERVE_KEY_PATH is deprecated, use SOFT_SERVE_DATA_PATH instead.")
		migrateWarn = true
	}
	if cfg.ReposPath != "" {
		log.Printf("warning: SOFT_SERVE_REPO_PATH is deprecated, use SOFT_SERVE_DATA_PATH instead.")
		migrateWarn = true
	}
	if migrateWarn {
		log.Printf("warning: please run `soft serve --migrate` to migrate your server and configuration.")
	}
	var db db.Store
	switch cfg.Db.Driver {
	case "sqlite":
		if err := os.MkdirAll(filepath.Join(cfg.DataPath, "db"), 0755); err != nil {
			log.Fatalln(err)
		}
		db, err = sqlite.New(filepath.Join(cfg.DataPath, "db", "soft-serve.db"))
		if err != nil {
			log.Fatalln(err)
		}
	}
	return cfg.WithDB(db)
}

// DB returns the database for the configuration.
func (c *Config) DB() db.Store {
	return c.db
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

// WithDB sets the database for the configuration.
func (c *Config) WithDB(db db.Store) *Config {
	c.db = db
	return c
}
