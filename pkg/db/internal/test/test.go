// Package test provides testing utilities for database operations.
package test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/soft-serve/pkg/db"
)

// OpenSqlite opens a new temp SQLite database for testing.
// It removes the database file when the test is done using tb.Cleanup.
// If ctx is nil, context.TODO() is used.
func OpenSqlite(ctx context.Context, tb testing.TB) (*db.DB, error) {
	if ctx == nil {
		ctx = context.TODO()
	}
	dbpath := filepath.Join(tb.TempDir(), "test.db")
	dbx, err := db.Open(ctx, "sqlite", dbpath)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}
	tb.Cleanup(func() {
		if err := dbx.Close(); err != nil {
			tb.Error(err)
		}
	})
	return dbx, nil
}
