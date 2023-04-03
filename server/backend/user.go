package backend

import (
	"golang.org/x/crypto/ssh"
)

// User is an interface representing a user.
type User interface {
	// Username returns the user's username.
	Username() string
	// IsAdmin returns whether the user is an admin.
	IsAdmin() bool
	// PublicKeys returns the user's public keys.
	PublicKeys() []ssh.PublicKey
}

// UserAccess is an interface that handles user access to repositories.
type UserAccess interface {
	// AccessLevel returns the access level of the username to the repository.
	AccessLevel(repo string, username string) AccessLevel
	// AccessLevelByPublicKey returns the access level of the public key to the repository.
	AccessLevelByPublicKey(repo string, pk ssh.PublicKey) AccessLevel
}

// UserStore is an interface for managing users.
type UserStore interface {
	// User finds the given user.
	User(username string) (User, error)
	// UserByPublicKey finds the user with the given public key.
	UserByPublicKey(pk ssh.PublicKey) (User, error)
	// Users returns a list of all users.
	Users() ([]string, error)
	// CreateUser creates a new user.
	CreateUser(username string, opts UserOptions) (User, error)
	// DeleteUser deletes a user.
	DeleteUser(username string) error
	// SetUsername sets the username of the user.
	SetUsername(oldUsername string, newUsername string) error
	// SetAdmin sets whether the user is an admin.
	SetAdmin(username string, admin bool) error
	// AddPublicKey adds a public key to the user.
	AddPublicKey(username string, pk ssh.PublicKey) error
	// RemovePublicKey removes a public key from the user.
	RemovePublicKey(username string, pk ssh.PublicKey) error
	// ListPublicKeys lists the public keys of the user.
	ListPublicKeys(username string) ([]ssh.PublicKey, error)
}

// UserOptions are options for creating a user.
type UserOptions struct {
	// Admin is whether the user is an admin.
	Admin bool
	// PublicKeys are the user's public keys.
	PublicKeys []ssh.PublicKey
}
