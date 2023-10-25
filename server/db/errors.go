package db

import (
	"database/sql"
	"errors"

	"github.com/lib/pq"
	sqlite "modernc.org/sqlite"
	sqlitelib "modernc.org/sqlite/lib"
)

var (
	// ErrDuplicateKey is a constraint violation error.
	ErrDuplicateKey = errors.New("duplicate key value violates table constraint")

	// ErrRecordNotFound is returned when a record is not found.
	ErrRecordNotFound = sql.ErrNoRows
)

// WrapError is a convenient function that unite various database driver
// errors to consistent errors.
func WrapError(err error) error {
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRecordNotFound
		}

		// Handle sqlite constraint error.
		if liteErr, ok := err.(*sqlite.Error); ok {
			code := liteErr.Code()
			if code == sqlitelib.SQLITE_CONSTRAINT_PRIMARYKEY ||
				code == sqlitelib.SQLITE_CONSTRAINT_FOREIGNKEY ||
				code == sqlitelib.SQLITE_CONSTRAINT_UNIQUE {
				return ErrDuplicateKey
			}
		}

		// Handle postgres constraint error.
		if pgErr, ok := err.(*pq.Error); ok {
			if pgErr.Code == "23505" ||
				pgErr.Code == "23503" ||
				pgErr.Code == "23514" {
				return ErrDuplicateKey
			}
		}
	}
	return err
}
