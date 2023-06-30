package config

// CacheConfig is the configuration for the cache server.
type CacheConfig struct {
	// Backend is the cache backend.
	Backend string `env:"CACHE_BACKEND" yaml:"backend"`
}

// Environ returns the environment variables for the cache configuration.
func (c CacheConfig) Environ() []string {
	return []string{
		"SOFT_SERVE_CACHE_BACKEND=" + c.Backend,
	}
}
