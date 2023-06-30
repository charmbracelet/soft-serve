package settings

import "context"

var contextKey = &struct{ string }{"settings"}

// FromContext returns the settings from the context.
func FromContext(ctx context.Context) Settings {
	if settings, ok := ctx.Value(contextKey).(Settings); ok {
		return settings
	}
	return nil
}

// WithContext returns a new context with the settings attached.
func WithContext(ctx context.Context, settings Settings) context.Context {
	return context.WithValue(ctx, contextKey, settings)
}
