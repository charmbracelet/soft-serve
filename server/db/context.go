package db

import "context"

var contextKey = struct{ string }{"db"}

// FromContext returns the database from the context.
func FromContext(ctx context.Context) *DB {
	if db, ok := ctx.Value(contextKey).(*DB); ok {
		return db
	}
	return nil
}

// WithContext returns a new context with the database.
func WithContext(ctx context.Context, db *DB) context.Context {
	return context.WithValue(ctx, contextKey, db)
}
