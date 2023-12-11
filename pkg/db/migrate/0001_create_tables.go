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
	Migrate: func(ctx context.Context, h db.Handler) error {
		cfg := config.FromContext(ctx)

		insert := "INSERT "

		// Alter old tables (if exist)
		// This is to support prior versions of Soft Serve v0.6
		switch h.DriverName() {
		case "sqlite3", "sqlite":
			insert += "OR IGNORE "

			hasUserTable := hasTable(h, "user")
			if hasUserTable {
				if _, err := h.ExecContext(ctx, "ALTER TABLE user RENAME TO user_old"); err != nil {
					return err
				}
			}

			if hasTable(h, "public_key") {
				if _, err := h.ExecContext(ctx, "ALTER TABLE public_key RENAME TO public_key_old"); err != nil {
					return err
				}
			}

			if hasTable(h, "collab") {
				if _, err := h.ExecContext(ctx, "ALTER TABLE collab RENAME TO collab_old"); err != nil {
					return err
				}
			}

			if hasTable(h, "repo") {
				if _, err := h.ExecContext(ctx, "ALTER TABLE repo RENAME TO repo_old"); err != nil {
					return err
				}
			}
		}

		if err := migrateUp(ctx, h, createTablesVersion, createTablesName); err != nil {
			return err
		}

		switch h.DriverName() {
		case "sqlite3", "sqlite":

			if _, err := h.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
				return err
			}

			if hasTable(h, "user_old") {
				sqlm := `
				INSERT INTO users (id, username, admin, updated_at)
					SELECT id, username, admin, updated_at FROM user_old;
				`
				if _, err := h.ExecContext(ctx, sqlm); err != nil {
					return err
				}
			}

			if hasTable(h, "public_key_old") {
				// Check duplicate keys
				pks := []struct {
					ID        string `db:"id"`
					PublicKey string `db:"public_key"`
				}{}
				if err := h.SelectContext(ctx, &pks, "SELECT id, public_key FROM public_key_old"); err != nil {
					return err
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
				if _, err := h.ExecContext(ctx, sqlm); err != nil {
					return err
				}
			}

			if hasTable(h, "repo_old") {
				sqlm := `
				INSERT INTO repos (id, name, project_name, description, private,mirror, hidden, created_at, updated_at, user_id)
					SELECT id, name, project_name, description, private, mirror, hidden, created_at, updated_at, (
						SELECT id FROM users WHERE admin = true ORDER BY id LIMIT 1
				) FROM repo_old;
				`
				if _, err := h.ExecContext(ctx, sqlm); err != nil {
					return err
				}
			}

			if hasTable(h, "collab_old") {
				sqlm := `
				INSERT INTO collabs (id, user_id, repo_id, access_level, created_at, updated_at)
					SELECT id, user_id, repo_id, ` + strconv.Itoa(int(access.ReadWriteAccess)) + `, created_at, updated_at FROM collab_old;
				`
				if _, err := h.ExecContext(ctx, sqlm); err != nil {
					return err
				}
			}

			if _, err := h.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
				return err
			}
		}

		// Insert default user
		insertUser := h.Rebind(insert + "INTO users (username, admin, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)")
		if _, err := h.ExecContext(ctx, insertUser, "admin", true); err != nil {
			return err
		}

		for _, k := range cfg.AdminKeys() {
			query := insert + "INTO public_keys (user_id, public_key, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)"
			if h.DriverName() == "postgres" {
				query += " ON CONFLICT DO NOTHING"
			}

			query = h.Rebind(query)
			ak := sshutils.MarshalAuthorizedKey(k)
			if _, err := h.ExecContext(ctx, query, 1, ak); err != nil {
				if errors.Is(db.WrapError(err), db.ErrDuplicateKey) {
					continue
				}
				return err
			}
		}

		// Insert default settings
		insertSettings := insert + "INTO settings (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)"
		insertSettings = h.Rebind(insertSettings)
		settings := []struct {
			Key   string
			Value string
		}{
			{"allow_keyless", "true"},
			{"anon_access", access.ReadOnlyAccess.String()},
			{"init", "true"},
		}

		for _, s := range settings {
			if _, err := h.ExecContext(ctx, insertSettings, s.Key, s.Value); err != nil {
				return fmt.Errorf("inserting default settings %q: %w", s.Key, err)
			}
		}

		return nil
	},
	Rollback: func(ctx context.Context, h db.Handler) error {
		return migrateDown(ctx, h, createTablesVersion, createTablesName)
	},
}
