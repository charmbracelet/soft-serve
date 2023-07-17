package backend

import "context"

// ContextKey is the key for the backend in the context.
var ContextKey = &struct{ string }{"backend"}

// FromContext returns the backend from a context.
func FromContext(ctx context.Context) *Backend {
	if b, ok := ctx.Value(ContextKey).(*Backend); ok {
		return b
	}

	return nil
}

// WithContext returns a new context with the backend attached.
func WithContext(ctx context.Context, b *Backend) context.Context {
	return context.WithValue(ctx, ContextKey, b)
}
