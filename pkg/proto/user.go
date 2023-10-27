package proto

import "golang.org/x/crypto/ssh"

// User is an interface representing a user.
type User interface {
	// ID returns the user's ID.
	ID() int64
	// Username returns the user's username.
	Username() string
	// IsAdmin returns whether the user is an admin.
	IsAdmin() bool
	// PublicKeys returns the user's public keys.
	PublicKeys() []ssh.PublicKey
	// Password returns the user's password hash.
	Password() string
}

// UserOptions are options for creating a user.
type UserOptions struct {
	// Admin is whether the user is an admin.
	Admin bool
	// PublicKeys are the user's public keys.
	PublicKeys []ssh.PublicKey
}
