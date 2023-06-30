package access

import "context"

var (
	contextKey            = &struct{ string }{"access"}
	ContextKeyAccessLevel = &struct{ string }{"access-level"}
)

// FromContext returns the access from the context.
func FromContext(ctx context.Context) Access {
	if access, ok := ctx.Value(contextKey).(Access); ok {
		return access
	}
	return nil
}

// WithContext returns a new context with the access attached.
func WithContext(ctx context.Context, access Access) context.Context {
	return context.WithValue(ctx, contextKey, access)
}

// AccessLevelFromContext returns the access level from the context.
func AccessLevelFromContext(ctx context.Context) AccessLevel {
	if al, ok := ctx.Value(ContextKeyAccessLevel).(AccessLevel); ok {
		return al
	}
	return NoAccess
}

// WithAccessLevelContext returns a new context with the access level attached.
func WithAccessLevelContext(ctx context.Context, al AccessLevel) context.Context {
	return context.WithValue(ctx, ContextKeyAccessLevel, al)
}
