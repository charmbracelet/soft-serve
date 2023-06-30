package sqlite

import (
	"github.com/charmbracelet/soft-serve/server/auth"
	"golang.org/x/crypto/ssh"
)

// UserOptions are options for creating a user.
type UserOptions struct {
	// Admin is whether the user is an admin.
	Admin bool
	// PublicKeys are the user's public keys.
	PublicKeys []ssh.PublicKey
}

// User represents a user.
type User struct {
	username   string
	isAdmin    bool
	publicKeys []ssh.PublicKey
}

var _ auth.User = (*User)(nil)

// IsAdmin implements auth.User.
func (u *User) IsAdmin() bool {
	return u.isAdmin
}

// PublicKeys implements auth.User.
func (u *User) PublicKeys() []ssh.PublicKey {
	if u.publicKeys == nil {
		return []ssh.PublicKey{}
	}

	return u.publicKeys
}

// Username implements auth.User.
func (u *User) Username() string {
	return u.username
}
