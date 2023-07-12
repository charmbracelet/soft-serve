package backend

import "context"

var contextKey = &struct{ string }{"backend"}

// FromContext returns the backend from a context.
func FromContext(ctx context.Context) *Backend {
	if b, ok := ctx.Value(contextKey).(*Backend); ok {
		return b
	}

	return nil
}

// WithContext returns a new context with the backend attached.
func WithContext(ctx context.Context, b *Backend) context.Context {
	return context.WithValue(ctx, contextKey, b)
}
