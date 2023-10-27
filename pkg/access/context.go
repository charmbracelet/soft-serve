package access

import "context"

// ContextKey is the context key for the access level.
var ContextKey = &struct{ string }{"access"}

// FromContext returns the access level from the context.
func FromContext(ctx context.Context) AccessLevel {
	if ac, ok := ctx.Value(ContextKey).(AccessLevel); ok {
		return ac
	}

	return -1
}

// WithContext returns a new context with the access level.
func WithContext(ctx context.Context, ac AccessLevel) context.Context {
	return context.WithValue(ctx, ContextKey, ac)
}
