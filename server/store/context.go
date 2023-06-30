package store

import "context"

var contextKey = &struct{ string }{"store"}

// FromContext returns the store from the context.
func FromContext(ctx context.Context) Store {
	if store, ok := ctx.Value(contextKey).(Store); ok {
		return store
	}
	return nil
}

// WithContext returns a new context with the store attached.
func WithContext(ctx context.Context, store Store) context.Context {
	return context.WithValue(ctx, contextKey, store)
}
