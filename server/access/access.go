package access

// AccessLevel is the level of access allowed to a repo.
type AccessLevel int // nolint: golint

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

// String returns the string representation of the access level.
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
		return "unknown"
	}
}

// ParseAccessLevel parses an access level string.
func ParseAccessLevel(s string) AccessLevel {
	switch s {
	case "no-access":
		return NoAccess
	case "read-only":
		return ReadOnlyAccess
	case "read-write":
		return ReadWriteAccess
	case "admin-access":
		return AdminAccess
	default:
		return AccessLevel(-1)
	}
}
