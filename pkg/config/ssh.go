package config

import (
	"errors"

	"github.com/charmbracelet/keygen"
)

var (
	// ErrNilConfig is returned when a nil config is passed to a function.
	ErrNilConfig = errors.New("nil config")

	// ErrEmptySSHKeyPath is returned when the SSH key path is empty.
	ErrEmptySSHKeyPath = errors.New("empty SSH key path")
)

// KeyPair returns the server's SSH key pair.
func (c SSHConfig) KeyPair() (*keygen.SSHKeyPair, error) {
	return keygen.New(c.KeyPath, keygen.WithKeyType(keygen.Ed25519))
}

// KeyPair returns the server's SSH key pair.
func KeyPair(cfg *Config) (*keygen.SSHKeyPair, error) {
	if cfg == nil {
		return nil, ErrNilConfig
	}

	if cfg.SSH.KeyPath == "" {
		return nil, ErrEmptySSHKeyPath
	}

	return keygen.New(cfg.SSH.KeyPath, keygen.WithKeyType(keygen.Ed25519))
}
