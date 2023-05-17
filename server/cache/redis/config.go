package redis

import (
	"github.com/charmbracelet/soft-serve/server/config"
)

// Config is the configuration for the Redis cache.
type Config struct {
	// Addr is the Redis address [host][:port].
	Addr string `env:"ADDR" yaml:"addr"`
	// Username is the Redis username.
	Username string `env:"USERNAME" yaml:"username"`
	// Password is the Redis password.
	Password string `env:"PASSWORD" yaml:"password"`
	// DB is the Redis database.
	DB int `env:"DB" yaml:"db"`
}

// NewConfig returns a new Redis cache configuration.
// If path is empty, the default config path will be used.
//
// TODO: add support for TLS and other Redis options.
func NewConfig(path string) (*Config, error) {
	if path == "" {
		path = config.DefaultConfig().FilePath()
	}

	cfg := DefaultConfig()
	wrapper := &struct {
		Redis *Config `envPrefix:"REDIS_" yaml:"redis"`
	}{
		Redis: cfg,
	}

	if err := config.ParseConfig(wrapper, path); err != nil {
		return cfg, err
	}

	return cfg, nil
}

// DefaultConfig returns the default configuration for the Redis cache.
func DefaultConfig() *Config {
	return &Config{
		Addr: "localhost:6379",
	}
}
