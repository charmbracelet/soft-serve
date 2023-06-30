package db

import "context"

var contextKey = &struct{ string }{"db"}

// FromContext returns the database from the context.
func FromContext(ctx context.Context) Database {
	if db, ok := ctx.Value(contextKey).(Database); ok {
		return db
	}
	return nil
}

// WithContext returns a new context with the database attached.
func WithContext(ctx context.Context, db Database) context.Context {
	return context.WithValue(ctx, contextKey, db)
}
