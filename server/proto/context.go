package proto

import "context"

// ContextKeyRepository is the context key for the repository.
var ContextKeyRepository = &struct{ string }{"repository"}

// ContextKeyUser is the context key for the user.
var ContextKeyUser = &struct{ string }{"user"}

// RepositoryFromContext returns the repository from the context.
func RepositoryFromContext(ctx context.Context) Repository {
	if r, ok := ctx.Value(ContextKeyRepository).(Repository); ok {
		return r
	}
	return nil
}

// UserFromContext returns the user from the context.
func UserFromContext(ctx context.Context) User {
	if u, ok := ctx.Value(ContextKeyUser).(User); ok {
		return u
	}
	return nil
}

// WithRepositoryContext returns a new context with the repository.
func WithRepositoryContext(ctx context.Context, r Repository) context.Context {
	return context.WithValue(ctx, ContextKeyRepository, r)
}

// WithUserContext returns a new context with the user.
func WithUserContext(ctx context.Context, u User) context.Context {
	return context.WithValue(ctx, ContextKeyUser, u)
}
