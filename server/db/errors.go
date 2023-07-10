package db

import (
	"database/sql"
	"errors"

	sqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

var (
	// ErrDuplicateKey is a constraint violation error.
	ErrDuplicateKey = errors.New("duplicate key value violates table constraint")
)

// WrapError is a convenient function that unite various database driver
// errors to consistent errors.
func WrapError(err error) error {
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return err
		}
		// Handle sqlite constraint error.
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
