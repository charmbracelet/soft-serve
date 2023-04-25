package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/caarlos0/env/v7"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/server/backend"
	"gopkg.in/yaml.v3"
)

// SSHConfig is the configuration for the SSH server.
type SSHConfig struct {
	// ListenAddr is the address on which the SSH server will listen.
	ListenAddr string `env:"LISTEN_ADDR" yaml:"listen_addr"`

	// PublicURL is the public URL of the SSH server.
	PublicURL string `env:"PUBLIC_URL" yaml:"public_url"`

	// KeyPath is the path to the SSH server's private key.
	KeyPath string `env:"KEY_PATH" yaml:"key_path"`

	// ClientKeyPath is the path to the SSH server's client private key.
	ClientKeyPath string `env:"CLIENT_KEY_PATH" yaml:"client_key_path"`

	// InternalKeyPath is the path to the SSH server's internal private key.
	InternalKeyPath string `env:"INTERNAL_KEY_PATH" yaml:"internal_key_path"`

	// MaxTimeout is the maximum number of seconds a connection can take.
	MaxTimeout int `env:"MAX_TIMEOUT" yaml:"max_timeout`

	// IdleTimeout is the number of seconds a connection can be idle before it is closed.
	IdleTimeout int `env:"IDLE_TIMEOUT" yaml:"idle_timeout"`
}

// GitConfig is the Git daemon configuration for the server.
type GitConfig struct {
	// ListenAddr is the address on which the Git daemon will listen.
	ListenAddr string `env:"LISTEN_ADDR" yaml:"listen_addr"`

	// MaxTimeout is the maximum number of seconds a connection can take.
	MaxTimeout int `env:"MAX_TIMEOUT" yaml:"max_timeout"`

	// IdleTimeout is the number of seconds a connection can be idle before it is closed.
	IdleTimeout int `env:"IDLE_TIMEOUT" yaml:"idle_timeout"`

	// MaxConnections is the maximum number of concurrent connections.
	MaxConnections int `env:"MAX_CONNECTIONS" yaml:"max_connections"`
}

// HTTPConfig is the HTTP configuration for the server.
type HTTPConfig struct {
	// ListenAddr is the address on which the HTTP server will listen.
	ListenAddr string `env:"LISTEN_ADDR" yaml:"listen_addr"`

	// TLSKeyPath is the path to the TLS private key.
	TLSKeyPath string `env:"TLS_KEY_PATH" yaml:"tls_key_path"`

	// TLSCertPath is the path to the TLS certificate.
	TLSCertPath string `env:"TLS_CERT_PATH" yaml:"tls_cert_path"`

	// PublicURL is the public URL of the HTTP server.
	PublicURL string `env:"PUBLIC_URL" yaml:"public_url"`
}

// StatsConfig is the configuration for the stats server.
type StatsConfig struct {
	// ListenAddr is the address on which the stats server will listen.
	ListenAddr string `env:"LISTEN_ADDR" yaml:"listen_addr"`
}

// Config is the configuration for Soft Serve.
type Config struct {
	// Name is the name of the server.
	Name string `env:"NAME" yaml:"name"`

	// SSH is the configuration for the SSH server.
	SSH SSHConfig `envPrefix:"SSH_" yaml:"ssh"`

	// Git is the configuration for the Git daemon.
	Git GitConfig `envPrefix:"GIT_" yaml:"git"`

	// HTTP is the configuration for the HTTP server.
	HTTP HTTPConfig `envPrefix:"HTTP_" yaml:"http"`

	// Stats is the configuration for the stats server.
	Stats StatsConfig `envPrefix:"STATS_" yaml:"stats"`

	// InitialAdminKeys is a list of public keys that will be added to the list of admins.
	InitialAdminKeys []string `env:"INITIAL_ADMIN_KEYS" envSeparator:"\n" yaml:"initial_admin_keys"`

	// DataPath is the path to the directory where Soft Serve will store its data.
	DataPath string `env:"DATA_PATH" yaml:"-"`

	// Backend is the Git backend to use.
	Backend backend.Backend `yaml:"-"`

	// InternalPublicKey is the public key of the internal SSH key.
	InternalPublicKey string `yaml:"-"`

	// ClientPublicKey is the public key of the client SSH key.
	ClientPublicKey string `yaml:"-"`
}

func parseConfig(path string) (*Config, error) {
	dataPath := filepath.Dir(path)
	cfg := &Config{
		Name:     "Soft Serve",
		DataPath: dataPath,
		SSH: SSHConfig{
			ListenAddr:      ":23231",
			PublicURL:       "ssh://localhost:23231",
			KeyPath:         filepath.Join("ssh", "soft_serve_host_ed25519"),
			ClientKeyPath:   filepath.Join("ssh", "soft_serve_client_ed25519"),
			InternalKeyPath: filepath.Join("ssh", "soft_serve_internal_ed25519"),
			MaxTimeout:      0,
			IdleTimeout:     120,
		},
		Git: GitConfig{
			ListenAddr:     ":9418",
			MaxTimeout:     0,
			IdleTimeout:    3,
			MaxConnections: 32,
		},
		HTTP: HTTPConfig{
			ListenAddr: ":8080",
			PublicURL:  "http://localhost:8080",
		},
		Stats: StatsConfig{
			ListenAddr: ":8081",
		},
	}

	f, err := os.Open(path)
	if err != nil {
		return cfg, err
	}

	defer f.Close() // nolint: errcheck
	if err := yaml.NewDecoder(f).Decode(cfg); err != nil {
		return cfg, fmt.Errorf("decode config: %w", err)
	}

	// Override with environment variables
	if err := env.Parse(cfg, env.Options{
		Prefix: "SOFT_SERVE_",
	}); err != nil {
		return cfg, fmt.Errorf("parse environment variables: %w", err)
	}

	// Reset datapath to config dir.
	// This is necessary because the environment variable may be set to
	// a different directory.
	cfg.DataPath = dataPath

	return cfg, nil
}

// ParseConfig parses the configuration from the given file.
func ParseConfig(path string) (*Config, error) {
	cfg, err := parseConfig(path)
	if err != nil {
		return nil, err
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// WriteConfig writes the configuration to the given file.
func WriteConfig(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(newConfigFile(cfg)), 0o600) // nolint: errcheck
}

// DefaultConfig returns a Config with the values populated with the defaults
// or specified environment variables.
func DefaultConfig() *Config {
	dataPath := os.Getenv("SOFT_SERVE_DATA_PATH")
	if dataPath == "" {
		dataPath = "data"
	}

	cp := filepath.Join(dataPath, "config.yaml")
	cfg, err := parseConfig(cp)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Errorf("failed to parse config: %v", err)
	}

	// Write config if it doesn't exist
	if _, err := os.Stat(cp); os.IsNotExist(err) {
		if err := WriteConfig(cp, cfg); err != nil {
			log.Fatal("failed to write config", "err", err)
		}
	}

	if err := cfg.validate(); err != nil {
		log.Fatal(err)
	}

	return cfg
}

// WithBackend sets the backend for the configuration.
func (c *Config) WithBackend(backend backend.Backend) *Config {
	c.Backend = backend
	return c
}

func (c *Config) validate() error {
	// Use absolute paths
	if !filepath.IsAbs(c.DataPath) {
		dp, err := filepath.Abs(c.DataPath)
		if err != nil {
			return err
		}
		c.DataPath = dp
	}

	c.SSH.PublicURL = strings.TrimSuffix(c.SSH.PublicURL, "/")
	c.HTTP.PublicURL = strings.TrimSuffix(c.HTTP.PublicURL, "/")

	if c.SSH.KeyPath != "" && !filepath.IsAbs(c.SSH.KeyPath) {
		c.SSH.KeyPath = filepath.Join(c.DataPath, c.SSH.KeyPath)
	}

	if c.SSH.ClientKeyPath != "" && !filepath.IsAbs(c.SSH.ClientKeyPath) {
		c.SSH.ClientKeyPath = filepath.Join(c.DataPath, c.SSH.ClientKeyPath)
	}

	if c.SSH.InternalKeyPath != "" && !filepath.IsAbs(c.SSH.InternalKeyPath) {
		c.SSH.InternalKeyPath = filepath.Join(c.DataPath, c.SSH.InternalKeyPath)
	}

	if c.HTTP.TLSKeyPath != "" && !filepath.IsAbs(c.HTTP.TLSKeyPath) {
		c.HTTP.TLSKeyPath = filepath.Join(c.DataPath, c.HTTP.TLSKeyPath)
	}

	if c.HTTP.TLSCertPath != "" && !filepath.IsAbs(c.HTTP.TLSCertPath) {
		c.HTTP.TLSCertPath = filepath.Join(c.DataPath, c.HTTP.TLSCertPath)
	}

	return nil
}
