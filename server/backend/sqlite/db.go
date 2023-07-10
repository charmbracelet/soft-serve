package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/db"
)

// Close closes the database.
func (d *SqliteBackend) Close() error {
	return d.db.Close()
}

// init creates the database.
func (d *SqliteBackend) init() error {
	return d.db.TransactionContext(context.Background(), func(tx *db.Tx) error {
		if _, err := tx.Exec(sqlCreateSettingsTable); err != nil {
			return err
		}
		if _, err := tx.Exec(sqlCreateUserTable); err != nil {
			return err
		}
		if _, err := tx.Exec(sqlCreatePublicKeyTable); err != nil {
			return err
		}
		if _, err := tx.Exec(sqlCreateRepoTable); err != nil {
			return err
		}
		if _, err := tx.Exec(sqlCreateCollabTable); err != nil {
			return err
		}

		// Set default settings.
		if _, err := tx.Exec("INSERT OR IGNORE INTO settings (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)", "allow_keyless", true); err != nil {
			return err
		}
		if _, err := tx.Exec("INSERT OR IGNORE INTO settings (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)", "anon_access", backend.ReadOnlyAccess.String()); err != nil {
			return err
		}

		var init bool
		if err := tx.Get(&init, "SELECT value FROM settings WHERE key = 'init'"); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}

		// Create default user.
		if !init {
			r, err := tx.Exec("INSERT OR IGNORE INTO user (username, admin, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP);", "admin", true)
			if err != nil {
				return err
			}
			userID, err := r.LastInsertId()
			if err != nil {
				return err
			}

			// Add initial keys
			// Don't use cfg.AdminKeys since it also includes the internal key
			// used for internal api access.
			for _, k := range d.cfg.InitialAdminKeys {
				pk, _, err := backend.ParseAuthorizedKey(k)
				if err != nil {
					d.logger.Error("error parsing initial admin key, skipping", "key", k, "err", err)
					continue
				}

				stmt, err := tx.Prepare(`INSERT INTO public_key (user_id, public_key, updated_at)
					VALUES (?, ?, CURRENT_TIMESTAMP);`)
				if err != nil {
					return err
				}

				defer stmt.Close() // nolint: errcheck
				if _, err := stmt.Exec(userID, backend.MarshalAuthorizedKey(pk)); err != nil {
					return err
				}
			}
		}

		// set init flag
		if _, err := tx.Exec("INSERT OR IGNORE INTO settings (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)", "init", true); err != nil {
			return err
		}

		return nil
	})
}
