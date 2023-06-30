package backend

import (
	"context"

	"github.com/charmbracelet/soft-serve/server/access"
	"github.com/charmbracelet/soft-serve/server/auth"
	"github.com/charmbracelet/soft-serve/server/settings"
	"github.com/charmbracelet/soft-serve/server/store"
)

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
	ctx = settings.WithContext(ctx, b.Settings)
	ctx = store.WithContext(ctx, b.Store)
	ctx = access.WithContext(ctx, b.Access)
	ctx = auth.WithContext(ctx, b.Auth)
	ctx = context.WithValue(ctx, contextKey, b)
	return ctx
}
