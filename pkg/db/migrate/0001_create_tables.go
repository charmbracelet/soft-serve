// Package migrate provides database migration functionality.
package migrate

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/sshutils"
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
		// This is to support prior versions of Soft Serve v0.6
		switch tx.DriverName() {
		case "sqlite3", "sqlite":
			insert += "OR IGNORE "

			hasUserTable := hasTable(tx, "user")
			if hasUserTable {
				if _, err := tx.ExecContext(ctx, "ALTER TABLE user RENAME TO user_old"); err != nil {
					return err //nolint:wrapcheck
				}
			}

			if hasTable(tx, "public_key") {
				if _, err := tx.ExecContext(ctx, "ALTER TABLE public_key RENAME TO public_key_old"); err != nil {
					return err //nolint:wrapcheck
				}
			}

			if hasTable(tx, "collab") {
				if _, err := tx.ExecContext(ctx, "ALTER TABLE collab RENAME TO collab_old"); err != nil {
					return err //nolint:wrapcheck
				}
			}

			if hasTable(tx, "repo") {
				if _, err := tx.ExecContext(ctx, "ALTER TABLE repo RENAME TO repo_old"); err != nil {
					return err //nolint:wrapcheck
				}
			}
		}

		if err := migrateUp(ctx, tx, createTablesVersion, createTablesName); err != nil {
			return err
		}

		switch tx.DriverName() {
		case "sqlite3", "sqlite":

			if _, err := tx.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
				return err //nolint:wrapcheck
			}

			if hasTable(tx, "user_old") {
				sqlm := `
				INSERT INTO users (id, username, admin, updated_at)
					SELECT id, username, admin, updated_at FROM user_old;
				`
				if _, err := tx.ExecContext(ctx, sqlm); err != nil {
					return err //nolint:wrapcheck
				}
			}

			if hasTable(tx, "public_key_old") {
				// Check duplicate keys
				pks := []struct {
					ID        string `db:"id"`
					PublicKey string `db:"public_key"`
				}{}
				if err := tx.SelectContext(ctx, &pks, "SELECT id, public_key FROM public_key_old"); err != nil {
					return err //nolint:wrapcheck
				}

				pkss := map[string]struct{}{}
				for _, pk := range pks {
					if _, ok := pkss[pk.PublicKey]; ok {
						return fmt.Errorf("duplicate public key: %q, please remove the duplicate key and try again", pk.PublicKey)
					}
					pkss[pk.PublicKey] = struct{}{}
				}

				sqlm := `
				INSERT INTO public_keys (id, user_id, public_key, created_at, updated_at)
					SELECT id, user_id, public_key, created_at, updated_at FROM public_key_old;
				`
				if _, err := tx.ExecContext(ctx, sqlm); err != nil {
					return err //nolint:wrapcheck
				}
			}

			if hasTable(tx, "repo_old") {
				sqlm := `
				INSERT INTO repos (id, name, project_name, description, private,mirror, hidden, created_at, updated_at, user_id)
					SELECT id, name, project_name, description, private, mirror, hidden, created_at, updated_at, (
						SELECT id FROM users WHERE admin = true ORDER BY id LIMIT 1
				) FROM repo_old;
				`
				if _, err := tx.ExecContext(ctx, sqlm); err != nil {
					return err //nolint:wrapcheck
				}
			}

			if hasTable(tx, "collab_old") {
				sqlm := `
				INSERT INTO collabs (id, user_id, repo_id, access_level, created_at, updated_at)
					SELECT id, user_id, repo_id, ` + strconv.Itoa(int(access.ReadWriteAccess)) + `, created_at, updated_at FROM collab_old;
				`
				if _, err := tx.ExecContext(ctx, sqlm); err != nil {
					return err //nolint:wrapcheck
				}
			}

			if _, err := tx.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
				return err //nolint:wrapcheck
			}
		}

		// Insert default user
		insertUser := tx.Rebind(insert + "INTO users (username, admin, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)")
		if _, err := tx.ExecContext(ctx, insertUser, "admin", true); err != nil {
			return err //nolint:wrapcheck
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
				return err //nolint:wrapcheck
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
