package migrate

import (
	"context"
	"fmt"

	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/db"
	"github.com/charmbracelet/soft-serve/server/sshutils"
	"github.com/charmbracelet/soft-serve/server/store"
)

const (
	createTablesName    = "create tables"
	createTablesVersion = 1
)

var createTables = Migration{
	Version: createTablesVersion,
	Name:    createTablesName,
	Migrate: func(ctx context.Context, tx *db.Tx) error {
		cfg := config.FromContext(ctx)

		if err := migrateUp(ctx, tx, createTablesVersion, createTablesName); err != nil {
			return err
		}

		// Insert default settings
		insertSettings := "INSERT INTO settings (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)"
		insertSettings = tx.Rebind(insertSettings)
		settings := []struct {
			Key   string
			Value string
		}{
			{"allow_keyless", "true"},
			{"anon_access", store.ReadOnlyAccess.String()},
			{"init", "true"},
		}

		for _, s := range settings {
			if _, err := tx.ExecContext(ctx, insertSettings, s.Key, s.Value); err != nil {
				return fmt.Errorf("inserting default settings %q: %w", s.Key, err)
			}
		}

		// Insert default user
		insertUser := tx.Rebind("INSERT INTO user (username, admin, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)")
		if _, err := tx.ExecContext(ctx, insertUser, "admin", true); err != nil {
			return err
		}

		for _, k := range cfg.AdminKeys() {
			ak := sshutils.MarshalAuthorizedKey(k)
			if _, err := tx.ExecContext(ctx, "INSERT INTO public_key (user_id, public_key, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)", 1, ak); err != nil {
				return err
			}
		}

		return nil
	},
	Rollback: func(ctx context.Context, tx *db.Tx) error {
		return migrateDown(ctx, tx, createTablesVersion, createTablesName)
	},
}
