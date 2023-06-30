package sqlite

import (
	"context"

	"github.com/charmbracelet/soft-serve/server/access"
	"github.com/charmbracelet/soft-serve/server/db"
	"github.com/charmbracelet/soft-serve/server/db/sqlite"
	"github.com/charmbracelet/soft-serve/server/settings"
	"github.com/jmoiron/sqlx"
)

func init() {
	settings.Register("sqlite", newSettings)
}

// SqliteSettings is a SQLite settings store.
type SqliteSettings struct {
	db  db.Database
	ctx context.Context
}

func newSettings(ctx context.Context) (settings.Settings, error) {
	sdb := db.FromContext(ctx)
	if sdb == nil {
		return nil, db.ErrNoDatabase
	}

	return &SqliteSettings{
		db:  sdb,
		ctx: ctx,
	}, nil
}

// AllowKeyless returns whether or not keyless access is allowed.
//
// It implements backend.Backend.
func (d *SqliteSettings) AllowKeyless(ctx context.Context) bool {
	var allow bool
	if err := sqlite.WrapTx(d.db.DBx(), ctx, func(tx *sqlx.Tx) error {
		return tx.Get(&allow, "SELECT value FROM settings WHERE key = ?;", "allow_keyless")
	}); err != nil {
		return false
	}

	return allow
}

// AnonAccess returns the level of anonymous access.
//
// It implements backend.Backend.
func (d *SqliteSettings) AnonAccess(ctx context.Context) access.AccessLevel {
	var level string
	if err := sqlite.WrapTx(d.db.DBx(), ctx, func(tx *sqlx.Tx) error {
		return tx.Get(&level, "SELECT value FROM settings WHERE key = ?;", "anon_access")
	}); err != nil {
		return access.NoAccess
	}

	return access.ParseAccessLevel(level)
}

// SetAllowKeyless sets whether or not keyless access is allowed.
//
// It implements backend.Backend.
func (d *SqliteSettings) SetAllowKeyless(ctx context.Context, allow bool) error {
	return sqlite.WrapDbErr(
		sqlite.WrapTx(d.db.DBx(), ctx, func(tx *sqlx.Tx) error {
			_, err := tx.Exec("UPDATE settings SET value = ?, updated_at = CURRENT_TIMESTAMP WHERE key = ?;", allow, "allow_keyless")
			return err
		}),
	)
}

// SetAnonAccess sets the level of anonymous access.
//
// It implements backend.Backend.
func (d *SqliteSettings) SetAnonAccess(ctx context.Context, level access.AccessLevel) error {
	return sqlite.WrapDbErr(
		sqlite.WrapTx(d.db.DBx(), ctx, func(tx *sqlx.Tx) error {
			_, err := tx.Exec("UPDATE settings SET value = ?, updated_at = CURRENT_TIMESTAMP WHERE key = ?;", level.String(), "anon_access")
			return err
		}),
	)
}
