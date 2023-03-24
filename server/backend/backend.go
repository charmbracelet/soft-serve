package backend

import (
	"bytes"

	"golang.org/x/crypto/ssh"
)

// Backend is an interface that handles repositories management and any
// non-Git related operations.
type Backend interface {
	// ServerName returns the server's name.
	ServerName() string
	// SetServerName sets the server's name.
	SetServerName(name string) error
	// ServerHost returns the server's host.
	ServerHost() string
	// SetServerHost sets the server's host.
	SetServerHost(host string) error
	// ServerPort returns the server's port.
	ServerPort() string
	// SetServerPort sets the server's port.
	SetServerPort(port string) error

	// AnonAccess returns the access level for anonymous users.
	AnonAccess() AccessLevel
	// SetAnonAccess sets the access level for anonymous users.
	SetAnonAccess(level AccessLevel) error
	// AllowKeyless returns true if keyless access is allowed.
	AllowKeyless() bool
	// SetAllowKeyless sets whether or not keyless access is allowed.
	SetAllowKeyless(allow bool) error

	// Repository finds the given repository.
	Repository(repo string) (Repository, error)
	// Repositories returns a list of all repositories.
	Repositories() ([]Repository, error)
	// CreateRepository creates a new repository.
	CreateRepository(name string, private bool) (Repository, error)
	// DeleteRepository deletes a repository.
	DeleteRepository(name string) error
	// RenameRepository renames a repository.
	RenameRepository(oldName, newName string) error

	// Description returns the repo's description.
	Description(repo string) string
	// SetDescription sets the repo's description.
	SetDescription(repo, desc string) error
	// IsPrivate returns true if the repository is private.
	IsPrivate(repo string) bool
	// SetPrivate sets the repository's private status.
	SetPrivate(repo string, priv bool) error

	// IsCollaborator returns true if the authorized key is a collaborator on the repository.
	IsCollaborator(pk ssh.PublicKey, repo string) bool
	// AddCollaborator adds the authorized key as a collaborator on the repository.
	AddCollaborator(pk ssh.PublicKey, repo string) error
	// IsAdmin returns true if the authorized key is an admin.
	IsAdmin(pk ssh.PublicKey) bool
	// AddAdmin adds the authorized key as an admin.
	AddAdmin(pk ssh.PublicKey) error
}

// ParseAuthorizedKey parses an authorized key string into a public key.
func ParseAuthorizedKey(ak string) (ssh.PublicKey, string, error) {
	pk, c, _, _, err := ssh.ParseAuthorizedKey([]byte(ak))
	return pk, c, err
}

// MarshalAuthorizedKey marshals a public key into an authorized key string.
//
// This is the inverse of ParseAuthorizedKey.
// This function is a copy of ssh.MarshalAuthorizedKey, but without the trailing newline.
// It returns an empty string if pk is nil.
func MarshalAuthorizedKey(pk ssh.PublicKey) string {
	if pk == nil {
		return ""
	}
	return string(bytes.TrimSuffix(ssh.MarshalAuthorizedKey(pk), []byte("\n")))
}
