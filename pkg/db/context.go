// Package db provides database interface and connection management for Soft Serve.
package db

import "context"

// ContextKey is the key used to store the database in the context.
var ContextKey = struct{ string }{"db"}

// FromContext returns the database from the context.
func FromContext(ctx context.Context) *DB {
	if db, ok := ctx.Value(ContextKey).(*DB); ok {
		return db
	}
	return nil
}

// WithContext returns a new context with the database.
func WithContext(ctx context.Context, db *DB) context.Context {
	return context.WithValue(ctx, ContextKey, db)
}
