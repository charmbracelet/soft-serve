package config

import "context"

var ContextKeyConfig = &struct{ string }{"config"}

// WithContext returns a new context with the configuration attached.
func WithContext(ctx context.Context, cfg *Config) context.Context {
	return context.WithValue(ctx, ContextKeyConfig, cfg)
}

// FromContext returns the configuration from the context.
func FromContext(ctx context.Context) *Config {
	if c, ok := ctx.Value(ContextKeyConfig).(*Config); ok {
		return c
	}

	return DefaultConfig()
}
