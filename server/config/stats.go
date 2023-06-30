package config

// StatsConfig is the configuration for the stats server.
type StatsConfig struct {
	// ListenAddr is the address on which the stats server will listen.
	ListenAddr string `env:"LISTEN_ADDR" yaml:"listen_addr"`
}

// Environ returns the environment variables for the config.
func (s StatsConfig) Environ() []string {
	return []string{
		"SOFT_SERVE_STATS_LISTEN_ADDR=" + s.ListenAddr,
	}
}
