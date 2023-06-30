package config

import "strconv"

// GitDaemonConfig is the Git daemon configuration for the server.
type GitDaemonConfig struct {
	// ListenAddr is the address on which the Git daemon will listen.
	ListenAddr string `env:"LISTEN_ADDR" yaml:"listen_addr"`

	// MaxTimeout is the maximum number of seconds a connection can take.
	MaxTimeout int `env:"MAX_TIMEOUT" yaml:"max_timeout"`

	// IdleTimeout is the number of seconds a connection can be idle before it is closed.
	IdleTimeout int `env:"IDLE_TIMEOUT" yaml:"idle_timeout"`

	// MaxConnections is the maximum number of concurrent connections.
	MaxConnections int `env:"MAX_CONNECTIONS" yaml:"max_connections"`
}

// Environ returns the environment variables for the config.
func (g GitDaemonConfig) Environ() []string {
	return []string{
		"SOFT_SERVE_GIT_DAEMON_LISTEN_ADDR=" + g.ListenAddr,
		"SOFT_SERVE_GIT_DAEMON_MAX_TIMEOUT=" + strconv.Itoa(g.MaxTimeout),
		"SOFT_SERVE_GIT_DAEMON_IDLE_TIMEOUT=" + strconv.Itoa(g.IdleTimeout),
		"SOFT_SERVE_GIT_DAEMON_MAX_CONNECTIONS=" + strconv.Itoa(g.MaxConnections),
	}
}
