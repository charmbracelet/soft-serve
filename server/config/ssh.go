package config

import "github.com/charmbracelet/keygen"

// KeyPair returns the server's SSH key pair.
func (c SSHConfig) KeyPair() (*keygen.SSHKeyPair, error) {
	return keygen.New(c.KeyPath, keygen.WithKeyType(keygen.Ed25519))
}
