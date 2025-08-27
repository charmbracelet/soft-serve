package migrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	log "github.com/charmbracelet/log/v2"
	"github.com/charmbracelet/soft-serve/pkg/db"
)

// MigrateFunc is a function that executes a migration.
type MigrateFunc func(ctx context.Context, tx *db.Tx) error //nolint:revive

// Migration is a struct that contains the name of the migration and the
// function to execute it.
type Migration struct {
	Version  int64
	Name     string
	Migrate  MigrateFunc
	Rollback MigrateFunc
}

// Migrations is a database model to store migrations.
type Migrations struct {
	ID      int64  `db:"id"`
	Name    string `db:"name"`
	Version int64  `db:"version"`
}

func (Migrations) schema(driverName string) string {
	switch driverName {
	case driverSQLite3, driverSQLite:
		return `CREATE TABLE IF NOT EXISTS migrations (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL,
				version INTEGER NOT NULL UNIQUE
			);
		`
	case "postgres":
		return `CREATE TABLE IF NOT EXISTS migrations (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			version INTEGER NOT NULL UNIQUE
		);
	`
	case "mysql":
		return `CREATE TABLE IF NOT EXISTS migrations (
			id INT NOT NULL AUTO_INCREMENT,
			name TEXT NOT NULL,
			version INT NOT NULL,
			UNIQUE (version),
			PRIMARY KEY (id)
		);
	`
	default:
		panic("unknown driver")
	}
}

// Migrate runs the migrations.
func Migrate(ctx context.Context, dbx *db.DB) error {
	logger := log.FromContext(ctx).WithPrefix("migrate")
	return dbx.TransactionContext(ctx, func(tx *db.Tx) error {
		if !hasTable(tx, "migrations") {
			if _, err := tx.Exec(Migrations{}.schema(tx.DriverName())); err != nil {
				return err
			}
		}

		var migrs Migrations
		if err := tx.Get(&migrs, tx.Rebind("SELECT * FROM migrations ORDER BY version DESC LIMIT 1")); err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return err
			}
		}

		for _, m := range migrations {
			if m.Version <= migrs.Version {
				continue
			}

			logger.Infof("running migration %d. %s", m.Version, m.Name)
			if err := m.Migrate(ctx, tx); err != nil {
				return err
			}

			if _, err := tx.Exec(tx.Rebind("INSERT INTO migrations (name, version) VALUES (?, ?)"), m.Name, m.Version); err != nil {
				return err
			}
		}

		return nil
	})
}

// Rollback rolls back a migration.
func Rollback(ctx context.Context, dbx *db.DB) error {
	logger := log.FromContext(ctx).WithPrefix("migrate")
	return dbx.TransactionContext(ctx, func(tx *db.Tx) error {
		var migrs Migrations
		if err := tx.Get(&migrs, tx.Rebind("SELECT * FROM migrations ORDER BY version DESC LIMIT 1")); err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("there are no migrations to rollback: %w", err)
			}
		}

		if migrs.Version == 0 || len(migrations) < int(migrs.Version) {
			return fmt.Errorf("there are no migrations to rollback")
		}

		m := migrations[migrs.Version-1]
		logger.Infof("rolling back migration %d. %s", m.Version, m.Name)
		if err := m.Rollback(ctx, tx); err != nil {
			return err
		}

		if _, err := tx.Exec(tx.Rebind("DELETE FROM migrations WHERE version = ?"), migrs.Version); err != nil {
			return err
		}

		return nil
	})
}

func hasTable(tx *db.Tx, tableName string) bool {
	var query string
	switch tx.DriverName() {
	case driverSQLite3, driverSQLite:
		query = "SELECT name FROM sqlite_master WHERE type='table' AND name=?"
	case driverPostgres:
		fallthrough
	case "mysql":
		query = "SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' AND table_name = ?"
	}

	query = tx.Rebind(query)
	var name string
	err := tx.Get(&name, query, tableName)
	return err == nil
}
