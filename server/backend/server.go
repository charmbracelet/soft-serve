package backend

// ServerBackend is an interface that handles server configuration.
type ServerBackend interface {
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
}
