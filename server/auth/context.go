package auth

import "context"

var (
	contextKey     = &struct{ string }{"auth"}
	ContextKeyUser = &struct{ string }{"user"}
)

// FromContext returns the auth from the context.
func FromContext(ctx context.Context) Auth {
	if auth, ok := ctx.Value(contextKey).(Auth); ok {
		return auth
	}
	return nil
}

// WithContext returns a new context with the auth attached.
func WithContext(ctx context.Context, auth Auth) context.Context {
	return context.WithValue(ctx, contextKey, auth)
}

// UserFromContext returns the user from the context.
func UserFromContext(ctx context.Context) User {
	if u, ok := ctx.Value(ContextKeyUser).(User); ok {
		return u
	}
	return nil
}

// WithUserContext returns a new context with the user attached.
func WithUserContext(ctx context.Context, u User) context.Context {
	return context.WithValue(ctx, ContextKeyUser, u)
}
