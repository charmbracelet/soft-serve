package cache

import "context"

var contextKey = &struct{ string }{"cache"}

// WithContext returns a new context with the cache.
func WithContext(ctx context.Context, c Cache) context.Context {
	if c == nil {
		return ctx
	}
	return context.WithValue(ctx, contextKey, c)
}

// FromContext returns the cache from the context.
// If no cache is found, nil is returned.
func FromContext(ctx context.Context) Cache {
	c, ok := ctx.Value(contextKey).(Cache)
	if !ok {
		return nil
	}

	return c
}
