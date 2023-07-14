package config

import "context"

// ContextKey is the context key for the config.
var ContextKey = struct{ string }{"config"}

// WithContext returns a new context with the configuration attached.
func WithContext(ctx context.Context, cfg *Config) context.Context {
	return context.WithValue(ctx, ContextKey, cfg)
}

// FromContext returns the configuration from the context.
func FromContext(ctx context.Context) *Config {
	if c, ok := ctx.Value(ContextKey).(*Config); ok {
		return c
	}

	return DefaultConfig()
}
