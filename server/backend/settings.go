package backend

// SettingsBackend is an interface that handles server configuration.
type SettingsBackend interface {
	// AnonAccess returns the access level for anonymous users.
	AnonAccess() AccessLevel
	// SetAnonAccess sets the access level for anonymous users.
	SetAnonAccess(level AccessLevel) error
	// AllowKeyless returns true if keyless access is allowed.
	AllowKeyless() bool
	// SetAllowKeyless sets whether or not keyless access is allowed.
	SetAllowKeyless(allow bool) error
}
