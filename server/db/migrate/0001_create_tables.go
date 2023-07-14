package migrate

import (
	"context"
	"errors"
	"fmt"

	"github.com/charmbracelet/soft-serve/server/access"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/db"
	"github.com/charmbracelet/soft-serve/server/sshutils"
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

		insert := "INSERT "

		// Alter old tables (if exist)
		// This is to support prior versions of Soft Serve
		switch tx.DriverName() {
		case "sqlite3", "sqlite":
			insert += "OR IGNORE "

			hasUserTable := hasTable(tx, "user")
			if hasUserTable {
				if _, err := tx.ExecContext(ctx, "ALTER TABLE user RENAME TO users"); err != nil {
					return err
				}
			}

			if hasTable(tx, "public_key") {
				if _, err := tx.ExecContext(ctx, "ALTER TABLE public_key RENAME TO public_keys"); err != nil {
					return err
				}
			}

			if hasTable(tx, "collab") {
				if _, err := tx.ExecContext(ctx, "ALTER TABLE collab RENAME TO collabs"); err != nil {
					return err
				}
			}

			if hasTable(tx, "repo") {
				if _, err := tx.ExecContext(ctx, "ALTER TABLE repo RENAME TO repos"); err != nil {
					return err
				}
			}

			// Fix username being nullable
			if hasUserTable {
				sqlm := `
				PRAGMA foreign_keys = OFF;

				CREATE TABLE users_new (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					username TEXT NOT NULL UNIQUE,
					admin BOOLEAN NOT NULL,
					created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME NOT NULL
				);

				INSERT INTO users_new (username, admin, updated_at)
					SELECT username, admin, updated_at FROM users;

				DROP TABLE users;
				ALTER TABLE users_new RENAME TO users;

				PRAGMA foreign_keys = ON;
				`
				if _, err := tx.ExecContext(ctx, sqlm); err != nil {
					return err
				}
			}
		}

		if err := migrateUp(ctx, tx, createTablesVersion, createTablesName); err != nil {
			return err
		}

		// Insert default user
		insertUser := tx.Rebind(insert + "INTO users (username, admin, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)")
		if _, err := tx.ExecContext(ctx, insertUser, "admin", true); err != nil {
			return err
		}

		for _, k := range cfg.AdminKeys() {
			query := insert + "INTO public_keys (user_id, public_key, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)"
			if tx.DriverName() == "postgres" {
				query += " ON CONFLICT DO NOTHING"
			}

			query = tx.Rebind(query)
			ak := sshutils.MarshalAuthorizedKey(k)
			if _, err := tx.ExecContext(ctx, query, 1, ak); err != nil {
				if errors.Is(db.WrapError(err), db.ErrDuplicateKey) {
					continue
				}
				return err
			}
		}

		// Insert default settings
		insertSettings := insert + "INTO settings (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)"
		insertSettings = tx.Rebind(insertSettings)
		settings := []struct {
			Key   string
			Value string
		}{
			{"allow_keyless", "true"},
			{"anon_access", access.ReadOnlyAccess.String()},
			{"init", "true"},
		}

		for _, s := range settings {
			if _, err := tx.ExecContext(ctx, insertSettings, s.Key, s.Value); err != nil {
				return fmt.Errorf("inserting default settings %q: %w", s.Key, err)
			}
		}

		return nil
	},
	Rollback: func(ctx context.Context, tx *db.Tx) error {
		return migrateDown(ctx, tx, createTablesVersion, createTablesName)
	},
}
