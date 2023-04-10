package config

import (
	"os"
	"path/filepath"

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
	InitialAdminKeys []string `env:"INITIAL_ADMIN_KEY" envSeparator:"\n" yaml:"initial_admin_keys"`

	// DataPath is the path to the directory where Soft Serve will store its data.
	DataPath string `env:"DATA_PATH" yaml:"-"`

	// Backend is the Git backend to use.
	Backend backend.Backend `yaml:"-"`

	// InternalPublicKey is the public key of the internal SSH key.
	InternalPublicKey string `yaml:"-"`

	// ClientPublicKey is the public key of the client SSH key.
	ClientPublicKey string `yaml:"-"`
}

// ParseConfig parses the configuration from the given file.
func ParseConfig(path string) (*Config, error) {
	cfg := &Config{}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close() // nolint: errcheck
	if err := yaml.NewDecoder(f).Decode(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// WriteConfig writes the configuration to the given file.
func WriteConfig(path string, cfg *Config) error {
	return os.WriteFile(path, []byte(newConfigFile(cfg)), 0o600) // nolint: errcheck
}

// DefaultConfig returns a Config with the values populated with the defaults
// or specified environment variables.
func DefaultConfig() *Config {
	dataPath := os.Getenv("SOFT_SERVE_DATA_PATH")
	if dataPath == "" {
		dataPath = "data"
	}

	dp, _ := filepath.Abs(dataPath)
	if dp != "" {
		dataPath = dp
	}

	cfg := &Config{
		Name:     "Soft Serve",
		DataPath: dataPath,
		SSH: SSHConfig{
			ListenAddr:      ":23231",
			PublicURL:       "ssh://localhost:23231",
			KeyPath:         filepath.Join("ssh", "soft_serve_host"),
			ClientKeyPath:   filepath.Join("ssh", "soft_serve_client"),
			InternalKeyPath: filepath.Join("ssh", "soft_serve_internal"),
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
	cp := filepath.Join(cfg.DataPath, "config.yaml")
	f, err := os.Open(cp)
	if err == nil {
		defer f.Close() // nolint: errcheck
		if err := yaml.NewDecoder(f).Decode(cfg); err != nil {
			log.Error("failed to decode config", "err", err)
		}
	} else {
		defer func() {
			os.WriteFile(cp, []byte(newConfigFile(cfg)), 0o600) // nolint: errcheck
		}()
	}

	if err := env.Parse(cfg, env.Options{
		Prefix: "SOFT_SERVE_",
	}); err != nil {
		log.Fatal(err)
	}

	return cfg
}

// WithBackend sets the backend for the configuration.
func (c *Config) WithBackend(backend backend.Backend) *Config {
	c.Backend = backend
	return c
}
