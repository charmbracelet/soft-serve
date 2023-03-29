package backend

// ServerBackend is an interface that handles server configuration.
type ServerBackend interface {
	// AnonAccess returns the access level for anonymous users.
	AnonAccess() AccessLevel
	// SetAnonAccess sets the access level for anonymous users.
	SetAnonAccess(level AccessLevel) error
	// AllowKeyless returns true if keyless access is allowed.
	AllowKeyless() bool
	// SetAllowKeyless sets whether or not keyless access is allowed.
	SetAllowKeyless(allow bool) error
	// DefaultBranch returns the default branch for new repositories.
	DefaultBranch() string
	// SetDefaultBranch sets the default branch for new repositories.
	SetDefaultBranch(branch string) error
}
