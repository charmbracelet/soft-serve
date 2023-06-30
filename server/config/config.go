package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/caarlos0/env/v8"
	"github.com/charmbracelet/soft-serve/server/sshutils"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v3"
)

// Config is the configuration for Soft Serve.
type Config struct {
	// Name is the name of the server.
	Name string `env:"NAME" yaml:"name"`

	// SSH is the configuration for the SSH server.
	SSH SSHConfig `envPrefix:"SSH_" yaml:"ssh"`

	// GitDaemon is the configuration for the GitDaemon daemon.
	GitDaemon GitDaemonConfig `envPrefix:"GIT_DAEMON_" yaml:"git_daemon"`

	// HTTP is the configuration for the HTTP server.
	HTTP HTTPConfig `envPrefix:"HTTP_" yaml:"http"`

	// Stats is the configuration for the stats server.
	Stats StatsConfig `envPrefix:"STATS_" yaml:"stats"`

	// Log is the logger configuration.
	Log LogConfig `envPrefix:"LOG_" yaml:"log"`

	// Cache is the cache backend to use.
	Cache CacheConfig `env:"CACHE" yaml:"cache"`

	// Database is the database configuration.
	Database DatabaseConfig `envPrefix:"DATABASE_" yaml:"database"`

	// Backend is the backend to use.
	Backend BackendConfig `envPrefix:"BACKEND_" yaml:"backend"`

	// InitialAdminKeys is a list of public keys that will be added to the list of admins.
	InitialAdminKeys []string `env:"INITIAL_ADMIN_KEYS" envSeparator:"\n" yaml:"initial_admin_keys"`

	// DataPath is the path to the directory where Soft Serve will store its data.
	DataPath string `env:"DATA_PATH" yaml:"-"`
}

// Environ returns the config as a list of environment variables.
// TODO: use pointer receiver
func (c *Config) Environ() []string {
	envs := append([]string{},
		"SOFT_SERVE_NAME="+c.Name,
		"SOFT_SERVE_DATA_PATH="+c.DataPath,
		"SOFT_SERVE_INITIAL_ADMIN_KEYS="+strings.Join(c.InitialAdminKeys, "\n"),
	)

	envs = append(envs, c.SSH.Environ()...)
	envs = append(envs, c.GitDaemon.Environ()...)
	envs = append(envs, c.HTTP.Environ()...)
	envs = append(envs, c.Stats.Environ()...)
	envs = append(envs, c.Log.Environ()...)
	envs = append(envs, c.Cache.Environ()...)
	envs = append(envs, c.Database.Environ()...)
	envs = append(envs, c.Backend.Environ()...)

	return envs
}

func parseFile(v interface{}, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open config file: %w", err)
	}

	defer f.Close() // nolint: errcheck
	if err := yaml.NewDecoder(f).Decode(v); err != nil {
		return fmt.Errorf("decode config: %w", err)
	}

	return nil
}

func parseEnv(v interface{}) error {
	// Override with environment variables
	if err := env.ParseWithOptions(v, env.Options{
		Prefix: "SOFT_SERVE_",
	}); err != nil {
		return fmt.Errorf("parse environment variables: %w", err)
	}

	return nil
}

// ParseConfig parses the configuration file to server configuration.
func ParseConfig(c *Config, path string) error {
	return parseConfig(c, path)
}

func parseConfig(cfg *Config, path string) error {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	if path != "" {
		// TODO: make config aware of config.yaml path
		cfg.DataPath = filepath.Dir(path)
	}

	exist := cfg.Exist()
	if exist {
		if err := parseFile(cfg, cfg.FilePath()); err != nil {
			return err
		}
	}

	// Merge initial admin keys from both config file and environment variables.
	initialAdminKeys := append([]string{}, cfg.InitialAdminKeys...)

	if err := parseEnv(cfg); err != nil {
		return err
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

	if err := cfg.validate(); err != nil {
		return err
	}

	return nil
}

// DefaultConfig returns the default config.
func DefaultConfig() *Config {
	dataPath := os.Getenv("SOFT_SERVE_DATA_PATH")
	if dataPath == "" {
		dataPath = "data"
	}

	return &Config{
		Name:             "Soft Serve",
		DataPath:         dataPath,
		InitialAdminKeys: []string{},
		SSH: SSHConfig{
			ListenAddr:    ":23231",
			PublicURL:     "ssh://localhost:23231",
			KeyPath:       filepath.Join("ssh", "soft_serve_host_ed25519"),
			ClientKeyPath: filepath.Join("ssh", "soft_serve_client_ed25519"),
			MaxTimeout:    0,
			IdleTimeout:   0,
		},
		GitDaemon: GitDaemonConfig{
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
		Cache: CacheConfig{
			Backend: "lru",
		},
		Database: DatabaseConfig{
			Driver:     "sqlite",
			DataSource: "soft-serve.db",
		},
		Backend: BackendConfig{
			Settings: "sqlite",
			Access:   "sqlite",
			Auth:     "sqlite",
			Store:    "sqlite",
		},
	}
}

// FilePath returns the expected config file path.
func (c *Config) FilePath() string {
	return filepath.Join(c.DataPath, "config.yaml")
}

// Exist returns true if the configuration file exists.
func (c *Config) Exist() bool {
	_, err := os.Stat(c.FilePath())
	return err == nil
}

// ReadConfig parses the configuration file.
func (c *Config) ReadConfig() error {
	return parseConfig(c, c.FilePath())
}

// WriteConfig writes the configuration in the default path.
func (c *Config) WriteConfig() error {
	return WriteConfig(c)
}

// ReposPath returns the expected repositories path.
func (c *Config) ReposPath() string {
	return filepath.Join(c.DataPath, "repos")
}

// WriteConfig writes the configuration in the default path.
func WriteConfig(c *Config) error {
	if c == nil {
		return fmt.Errorf("nil config")
	}

	fp := c.FilePath()
	if err := os.MkdirAll(filepath.Dir(fp), os.ModePerm); err != nil {
		return err
	}

	return os.WriteFile(fp, []byte(newConfigFile(c)), 0o644) // nolint: errcheck
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

	if c.SSH.KeyPath != "" && !filepath.IsAbs(c.SSH.KeyPath) {
		c.SSH.KeyPath = filepath.Join(c.DataPath, c.SSH.KeyPath)
	}

	if c.SSH.ClientKeyPath != "" && !filepath.IsAbs(c.SSH.ClientKeyPath) {
		c.SSH.ClientKeyPath = filepath.Join(c.DataPath, c.SSH.ClientKeyPath)
	}

	c.HTTP.PublicURL = strings.TrimSuffix(c.HTTP.PublicURL, "/")

	if c.HTTP.TLSKeyPath != "" && !filepath.IsAbs(c.HTTP.TLSKeyPath) {
		c.HTTP.TLSKeyPath = filepath.Join(c.DataPath, c.HTTP.TLSKeyPath)
	}

	if c.HTTP.TLSCertPath != "" && !filepath.IsAbs(c.HTTP.TLSCertPath) {
		c.HTTP.TLSCertPath = filepath.Join(c.DataPath, c.HTTP.TLSCertPath)
	}

	switch c.Database.Driver {
	case "sqlite":
		if c.Database.DataSource != "" && !filepath.IsAbs(c.Database.DataSource) {
			c.Database.DataSource = filepath.Join(c.DataPath, c.Database.DataSource)
		}
	}

	return nil
}

// parseAuthKeys parses authorized keys from either file paths or string authorized_keys.
func parseAuthKeys(aks []string) []ssh.PublicKey {
	pks := make([]ssh.PublicKey, 0)
	for _, key := range aks {
		if bts, err := os.ReadFile(key); err == nil {
			// key is a file
			key = strings.TrimSpace(string(bts))
		}
		if pk, _, err := sshutils.ParseAuthorizedKey(key); err == nil {
			pks = append(pks, pk)
		}
	}
	return pks
}

// AdminKeys returns the server admin keys.
func (c *Config) AdminKeys() []ssh.PublicKey {
	if c.InitialAdminKeys == nil {
		return []ssh.PublicKey{}
	}

	log.Print(c.InitialAdminKeys)
	return parseAuthKeys(c.InitialAdminKeys)
}
