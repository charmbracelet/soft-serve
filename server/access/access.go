package access

import (
	"context"

	"github.com/charmbracelet/soft-serve/server/auth"
)

// Access is an interface that represents repository access.
type Access interface {
	// AccessLevel returns the access level for the given repo.
	AccessLevel(ctx context.Context, repo string, user auth.User) (AccessLevel, error)
}
