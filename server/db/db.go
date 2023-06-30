package db

import (
	"context"
	"errors"

	"github.com/jmoiron/sqlx"
)

var (
	// ErrNoDatabase is returned when no database is found in the context.
	ErrNoDatabase = errors.New("no database found")
)

// Database is the interface that wraps basic database operations.
type Database interface {
	// Close closes the database connection.
	Close() error
	// Open opens a new database connection.
	Open(ctx context.Context, url string) (Database, error)
	// Migrate runs database migrations.
	Migrate(url string) error
	// DBx returns the underlying sqlx database.
	DBx() *sqlx.DB
}
