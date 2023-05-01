package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/jmoiron/sqlx"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// Close closes the database.
func (d *SqliteBackend) Close() error {
	return d.db.Close()
}

// init creates the database.
func (d *SqliteBackend) init() error {
	return wrapTx(d.db, context.Background(), func(tx *sqlx.Tx) error {
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

func wrapDbErr(err error) error {
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNoRecord
		}
		if liteErr, ok := err.(*sqlite.Error); ok {
			code := liteErr.Code()
			if code == sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY ||
				code == sqlite3.SQLITE_CONSTRAINT_UNIQUE {
				return ErrDuplicateKey
			}
		}
	}
	return err
}

func wrapTx(db *sqlx.DB, ctx context.Context, fn func(tx *sqlx.Tx) error) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		return rollback(tx, err)
	}

	if err := tx.Commit(); err != nil {
		if errors.Is(err, sql.ErrTxDone) {
			// this is ok because whoever did finish the tx should have also written the error already.
			return nil
		}
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func rollback(tx *sqlx.Tx, err error) error {
	if rerr := tx.Rollback(); rerr != nil {
		if errors.Is(rerr, sql.ErrTxDone) {
			return err
		}
		return fmt.Errorf("failed to rollback: %s: %w", err.Error(), rerr)
	}

	return err
}
