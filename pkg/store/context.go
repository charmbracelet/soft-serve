package store

import "context"

// ContextKey is the store context key.
var ContextKey = &struct{ string }{"store"}

// FromContext returns the store from the given context.
func FromContext(ctx context.Context) Store {
	if s, ok := ctx.Value(ContextKey).(Store); ok {
		return s
	}

	return nil
}

// WithContext returns a new context with the given store.
func WithContext(ctx context.Context, s Store) context.Context {
	return context.WithValue(ctx, ContextKey, s)
}
