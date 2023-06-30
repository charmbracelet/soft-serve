package config

import (
	"strconv"
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

// Environ returns the environment variables for the config.
func (s SSHConfig) Environ() []string {
	return []string{
		"SOFT_SERVE_SSH_LISTEN_ADDR=" + s.ListenAddr,
		"SOFT_SERVE_SSH_PUBLIC_URL=" + s.PublicURL,
		"SOFT_SERVE_SSH_KEY_PATH=" + s.KeyPath,
		"SOFT_SERVE_SSH_CLIENT_KEY_PATH=" + s.ClientKeyPath,
		"SOFT_SERVE_SSH_MAX_TIMEOUT=" + strconv.Itoa(s.MaxTimeout),
		"SOFT_SERVE_SSH_IDLE_TIMEOUT=" + strconv.Itoa(s.IdleTimeout),
	}
}
