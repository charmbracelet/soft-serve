package backend

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/db"
)

// AllowKeyless returns whether or not keyless access is allowed.
//
// It implements backend.Backend.
func (b *Backend) AllowKeyless(ctx context.Context) bool {
	var allow bool
	if err := b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		allow, err = b.store.GetAllowKeylessAccess(ctx, tx)
		return err
	}); err != nil {
		return false
	}

	return allow
}

// SetAllowKeyless sets whether or not keyless access is allowed.
//
// It implements backend.Backend.
func (b *Backend) SetAllowKeyless(ctx context.Context, allow bool) error {
	return b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return b.store.SetAllowKeylessAccess(ctx, tx, allow)
	})
}

// AnonAccess returns the level of anonymous access.
//
// It implements backend.Backend.
func (b *Backend) AnonAccess(ctx context.Context) access.AccessLevel {
	var level access.AccessLevel
	if err := b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		level, err = b.store.GetAnonAccess(ctx, tx)
		return err
	}); err != nil {
		return access.NoAccess
	}

	return level
}

// SetAnonAccess sets the level of anonymous access.
//
// It implements backend.Backend.
func (b *Backend) SetAnonAccess(ctx context.Context, level access.AccessLevel) error {
	return b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return b.store.SetAnonAccess(ctx, tx, level)
	})
}
