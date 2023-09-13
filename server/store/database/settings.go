package database

import (
	"context"

	"github.com/charmbracelet/soft-serve/server/access"
	"github.com/charmbracelet/soft-serve/server/db"
	"github.com/charmbracelet/soft-serve/server/store"
)

type settingsStore struct{}

var _ store.SettingStore = (*settingsStore)(nil)

// GetAllowKeylessAccess implements store.SettingStore.
func (*settingsStore) GetAllowKeylessAccess(ctx context.Context, tx db.Handler) (bool, error) {
	var allow bool
	query := tx.Rebind(`SELECT value FROM settings WHERE "key" = 'allow_keyless'`)
	if err := tx.GetContext(ctx, &allow, query); err != nil {
		return false, db.WrapError(err)
	}
	return allow, nil
}

// GetAnonAccess implements store.SettingStore.
func (*settingsStore) GetAnonAccess(ctx context.Context, tx db.Handler) (access.AccessLevel, error) {
	var level string
	query := tx.Rebind(`SELECT value FROM settings WHERE "key" = 'anon_access'`)
	if err := tx.GetContext(ctx, &level, query); err != nil {
		return access.NoAccess, db.WrapError(err)
	}
	return access.ParseAccessLevel(level), nil
}

// SetAllowKeylessAccess implements store.SettingStore.
func (*settingsStore) SetAllowKeylessAccess(ctx context.Context, tx db.Handler, allow bool) error {
	query := tx.Rebind(`UPDATE settings SET value = ?, updated_at = CURRENT_TIMESTAMP WHERE "key" = 'allow_keyless'`)
	_, err := tx.ExecContext(ctx, query, allow)
	return db.WrapError(err)
}

// SetAnonAccess implements store.SettingStore.
func (*settingsStore) SetAnonAccess(ctx context.Context, tx db.Handler, level access.AccessLevel) error {
	query := tx.Rebind(`UPDATE settings SET value = ?, updated_at = CURRENT_TIMESTAMP WHERE "key" = 'anon_access'`)
	_, err := tx.ExecContext(ctx, query, level.String())
	return db.WrapError(err)
}
