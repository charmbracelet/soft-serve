package settings

import (
	"context"

	"github.com/charmbracelet/soft-serve/server/access"
)

// Settings is an interface that manage server settings.
type Settings interface {
	// AnonAccess returns the access level for anonymous users.
	AnonAccess(ctx context.Context) access.AccessLevel
	// SetAnonAccess sets the access level for anonymous users.
	SetAnonAccess(ctx context.Context, level access.AccessLevel) error
	// AllowKeyless returns true if keyless access is allowed.
	AllowKeyless(ctx context.Context) bool
	// SetAllowKeyless sets whether or not keyless access is allowed.
	SetAllowKeyless(ctx context.Context, allow bool) error
}
