package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/caarlos0/env/v8"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/server/sshutils"
	"golang.org/x/crypto/ssh"
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

	// ClientKeyPath is the path to the server's client private key.
	ClientKeyPath string `env:"CLIENT_KEY_PATH" yaml:"client_key_path"`

	// MaxTimeout is the maximum number of seconds a connection can take.
	MaxTimeout int `env:"MAX_TIMEOUT" yaml:"max_timeout"`

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

// LogConfig is the logger configuration.
type LogConfig struct {
	// Format is the format of the logs.
	// Valid values are "json", "logfmt", and "text".
	Format string `env:"FORMAT" yaml:"format"`

	// Time format for the log `ts` field.
	// Format must be described in Golang's time format.
	TimeFormat string `env:"TIME_FORMAT" yaml:"time_format"`
}

// DBConfig is the database connection configuration.
type DBConfig struct {
	// Driver is the driver for the database.
	Driver string `env:"DRIVER" yaml:"driver"`

	// DataSource is the database data source name.
	DataSource string `env:"DATA_SOURCE" yaml:"data_source"`
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

	// Log is the logger configuration.
	Log LogConfig `envPrefix:"LOG_" yaml:"log"`

	// DB is the database configuration.
	DB DBConfig `envPrefix:"DB_" yaml:"db"`

	// InitialAdminKeys is a list of public keys that will be added to the list of admins.
	InitialAdminKeys []string `env:"INITIAL_ADMIN_KEYS" envSeparator:"\n" yaml:"initial_admin_keys"`

	// DataPath is the path to the directory where Soft Serve will store its data.
	DataPath string `env:"DATA_PATH" yaml:"-"`
}

// Environ returns the config as a list of environment variables.
func (c *Config) Environ() []string {
	envs := []string{}
	if c == nil {
		return envs
	}

	// TODO: do this dynamically
	envs = append(envs, []string{
		fmt.Sprintf("SOFT_SERVE_NAME=%s", c.Name),
		fmt.Sprintf("SOFT_SERVE_DATA_PATH=%s", c.DataPath),
		fmt.Sprintf("SOFT_SERVE_INITIAL_ADMIN_KEYS=%s", strings.Join(c.InitialAdminKeys, "\n")),
		fmt.Sprintf("SOFT_SERVE_SSH_LISTEN_ADDR=%s", c.SSH.ListenAddr),
		fmt.Sprintf("SOFT_SERVE_SSH_PUBLIC_URL=%s", c.SSH.PublicURL),
		fmt.Sprintf("SOFT_SERVE_SSH_KEY_PATH=%s", c.SSH.KeyPath),
		fmt.Sprintf("SOFT_SERVE_SSH_CLIENT_KEY_PATH=%s", c.SSH.ClientKeyPath),
		fmt.Sprintf("SOFT_SERVE_SSH_MAX_TIMEOUT=%d", c.SSH.MaxTimeout),
		fmt.Sprintf("SOFT_SERVE_SSH_IDLE_TIMEOUT=%d", c.SSH.IdleTimeout),
		fmt.Sprintf("SOFT_SERVE_GIT_LISTEN_ADDR=%s", c.Git.ListenAddr),
		fmt.Sprintf("SOFT_SERVE_GIT_MAX_TIMEOUT=%d", c.Git.MaxTimeout),
		fmt.Sprintf("SOFT_SERVE_GIT_IDLE_TIMEOUT=%d", c.Git.IdleTimeout),
		fmt.Sprintf("SOFT_SERVE_GIT_MAX_CONNECTIONS=%d", c.Git.MaxConnections),
		fmt.Sprintf("SOFT_SERVE_HTTP_LISTEN_ADDR=%s", c.HTTP.ListenAddr),
		fmt.Sprintf("SOFT_SERVE_HTTP_TLS_KEY_PATH=%s", c.HTTP.TLSKeyPath),
		fmt.Sprintf("SOFT_SERVE_HTTP_TLS_CERT_PATH=%s", c.HTTP.TLSCertPath),
		fmt.Sprintf("SOFT_SERVE_HTTP_PUBLIC_URL=%s", c.HTTP.PublicURL),
		fmt.Sprintf("SOFT_SERVE_STATS_LISTEN_ADDR=%s", c.Stats.ListenAddr),
		fmt.Sprintf("SOFT_SERVE_LOG_FORMAT=%s", c.Log.Format),
		fmt.Sprintf("SOFT_SERVE_LOG_TIME_FORMAT=%s", c.Log.TimeFormat),
		fmt.Sprintf("SOFT_SERVE_DB_DRIVER=%s", c.DB.Driver),
		fmt.Sprintf("SOFT_SERVE_DB_DATA_SOURCE=%s", c.DB.DataSource),
	}...)

	return envs
}

// IsDebug returns true if the server is running in debug mode.
func IsDebug() bool {
	debug, _ := strconv.ParseBool(os.Getenv("SOFT_SERVE_DEBUG"))
	return debug
}

// IsVerbose returns true if the server is running in verbose mode.
// Verbose mode is only enabled if debug mode is enabled.
func IsVerbose() bool {
	verbose, _ := strconv.ParseBool(os.Getenv("SOFT_SERVE_VERBOSE"))
	return IsDebug() && verbose
}

func parseConfig(path string) (*Config, error) {
	dataPath := filepath.Dir(path)
	cfg := &Config{
		Name:     "Soft Serve",
		DataPath: dataPath,
		SSH: SSHConfig{
			ListenAddr:    ":23231",
			PublicURL:     "ssh://localhost:23231",
			KeyPath:       filepath.Join("ssh", "soft_serve_host_ed25519"),
			ClientKeyPath: filepath.Join("ssh", "soft_serve_client_ed25519"),
			MaxTimeout:    0,
			IdleTimeout:   0,
		},
		Git: GitConfig{
			ListenAddr:     ":9418",
			MaxTimeout:     0,
			IdleTimeout:    3,
			MaxConnections: 32,
		},
		HTTP: HTTPConfig{
			ListenAddr: ":23232",
			PublicURL:  "http://localhost:23232",
		},
		Stats: StatsConfig{
			ListenAddr: "localhost:23233",
		},
		Log: LogConfig{
			Format:     "text",
			TimeFormat: time.DateTime,
		},
		DB: DBConfig{
			Driver: "sqlite",
			DataSource: "soft-serve.db" +
				"?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)",
		},
	}

	f, err := os.Open(path)
	if err == nil {
		defer f.Close() // nolint: errcheck
		if err := yaml.NewDecoder(f).Decode(cfg); err != nil {
			return cfg, fmt.Errorf("decode config: %w", err)
		}
	}

	// Merge initial admin keys from both config file and environment variables.
	initialAdminKeys := append([]string{}, cfg.InitialAdminKeys...)

	// Override with environment variables
	if err := env.ParseWithOptions(cfg, env.Options{
		Prefix: "SOFT_SERVE_",
	}); err != nil {
		return cfg, fmt.Errorf("parse environment variables: %w", err)
	}

	// Merge initial admin keys from environment variables.
	if initialAdminKeysEnv := os.Getenv("SOFT_SERVE_INITIAL_ADMIN_KEYS"); initialAdminKeysEnv != "" {
		cfg.InitialAdminKeys = append(cfg.InitialAdminKeys, initialAdminKeys...)
	}

	// Validate keys
	pks := make([]string, 0)
	for _, key := range parseAuthKeys(cfg.InitialAdminKeys) {
		ak := sshutils.MarshalAuthorizedKey(key)
		pks = append(pks, ak)
	}

	cfg.InitialAdminKeys = pks

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
		return cfg, err
	}

	if err := cfg.validate(); err != nil {
		return cfg, err
	}

	return cfg, nil
}

// WriteConfig writes the configuration to the given file.
func WriteConfig(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(newConfigFile(cfg)), 0o644) // nolint: errcheck
}

// DataPath returns the path to the data directory.
func DataPath() string {
	dp := os.Getenv("SOFT_SERVE_DATA_PATH")
	if dp == "" {
		dp = "data"
	}

	return dp
}

// DefaultConfig returns a Config with the values populated with the defaults
// or specified environment variables.
func DefaultConfig() *Config {
	dataPath := DataPath()
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

	if c.HTTP.TLSKeyPath != "" && !filepath.IsAbs(c.HTTP.TLSKeyPath) {
		c.HTTP.TLSKeyPath = filepath.Join(c.DataPath, c.HTTP.TLSKeyPath)
	}

	if c.HTTP.TLSCertPath != "" && !filepath.IsAbs(c.HTTP.TLSCertPath) {
		c.HTTP.TLSCertPath = filepath.Join(c.DataPath, c.HTTP.TLSCertPath)
	}

	if strings.HasPrefix(c.DB.Driver, "sqlite") && !filepath.IsAbs(c.DB.DataSource) {
		c.DB.DataSource = filepath.Join(c.DataPath, c.DB.DataSource)
	}

	return nil
}

// parseAuthKeys parses authorized keys from either file paths or string authorized_keys.
func parseAuthKeys(aks []string) []ssh.PublicKey {
	exist := make(map[string]struct{}, 0)
	pks := make([]ssh.PublicKey, 0)
	for _, key := range aks {
		if bts, err := os.ReadFile(key); err == nil {
			// key is a file
			key = strings.TrimSpace(string(bts))
		}

		if pk, _, err := sshutils.ParseAuthorizedKey(key); err == nil {
			if _, ok := exist[key]; !ok {
				pks = append(pks, pk)
				exist[key] = struct{}{}
			}
		}
	}
	return pks
}

// AdminKeys returns the server admin keys.
func (c *Config) AdminKeys() []ssh.PublicKey {
	return parseAuthKeys(c.InitialAdminKeys)
}

var configCtxKey = struct{ string }{"config"}

// WithContext returns a new context with the configuration attached.
func WithContext(ctx context.Context, cfg *Config) context.Context {
	return context.WithValue(ctx, configCtxKey, cfg)
}

// FromContext returns the configuration from the context.
func FromContext(ctx context.Context) *Config {
	if c, ok := ctx.Value(configCtxKey).(*Config); ok {
		return c
	}

	return DefaultConfig()
}
