package proto

import (
	"fmt"
	"strings"

	"github.com/gliderlabs/ssh"
)

// AccessLevel is the level of access allowed to a repo.
type AccessLevel int

const (
	// NoAccess does not allow access to the repo.
	NoAccess AccessLevel = iota

	// ReadOnlyAccess allows read-only access to the repo.
	ReadOnlyAccess

	// ReadWriteAccess allows read and write access to the repo.
	ReadWriteAccess

	// AdminAccess allows read, write, and admin access to the repo.
	AdminAccess
)

// String implements the Stringer interface for AccessLevel.
func (a AccessLevel) String() string {
	switch a {
	case NoAccess:
		return "no-access"
	case ReadOnlyAccess:
		return "read-only"
	case ReadWriteAccess:
		return "read-write"
	case AdminAccess:
		return "admin-access"
	default:
		return ""
	}
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (a *AccessLevel) UnmarshalText(text []byte) error {
	switch strings.ToLower(string(text)) {
	case "no-access":
		*a = NoAccess
	case "read-only":
		*a = ReadOnlyAccess
	case "read-write":
		*a = ReadWriteAccess
	case "admin-access":
		*a = AdminAccess
	default:
		return fmt.Errorf("invalid access level: %s", text)
	}
	return nil
}

// Access is an interface that defines the access level for repositories.
type Access interface {
	AuthRepo(repo string, pk ssh.PublicKey) AccessLevel
	IsCollab(repo string, pk ssh.PublicKey) bool
	IsAdmin(pk ssh.PublicKey) bool
}
