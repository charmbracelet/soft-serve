package auth

import "golang.org/x/crypto/ssh"

// User is an interface representing a user.
type User interface {
	// Username returns the user's username.
	Username() string
	// IsAdmin returns whether the user is an admin.
	IsAdmin() bool
	// PublicKeys returns the user's public keys.
	PublicKeys() []ssh.PublicKey
}
