package settings

import (
	"context"
	"fmt"
	"sync"
)

var (
	registry = map[string]Constructor{}
	mtx      sync.RWMutex

	// ErrNotFound is returned when a setting is not found.
	ErrNotFound = fmt.Errorf("setting not found")
)

// Constructor is a function that returns a new setting.
type Constructor func(ctx context.Context) (Settings, error)

// Register registers a setting.
func Register(name string, fn Constructor) {
	mtx.Lock()
	defer mtx.Unlock()

	registry[name] = fn
}

// New returns a new setting.
func New(ctx context.Context, name string) (Settings, error) {
	mtx.RLock()
	fn, ok := registry[name]
	mtx.RUnlock()

	if !ok {
		return nil, ErrNotFound
	}

	return fn(ctx)
}

// List returns a list of registered settings.
func List() []string {
	mtx.Lock()
	defer mtx.Unlock()
	settings := make([]string, 0)
	for name := range registry {
		settings = append(settings, name)
	}
	return settings
}
