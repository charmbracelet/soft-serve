package access

import (
	"context"
	"errors"
	"sync"
)

var (
	registry = make(map[string]Constructor)
	mtx      sync.RWMutex

	// ErrNotFound is returned when an access provider is not found.
	ErrNotFound = errors.New("access provider not found")
)

// Constructor is a function that returns an access provider.
type Constructor func(ctx context.Context) (Access, error)

// Register registers an access provider.
func Register(name string, fn Constructor) {
	mtx.Lock()
	defer mtx.Unlock()
	registry[name] = fn
}

// New returns a new access provider.
func New(ctx context.Context, name string) (Access, error) {
	mtx.RLock()
	fn, ok := registry[name]
	mtx.RUnlock()

	if !ok {
		return nil, ErrNotFound
	}

	return fn(ctx)
}

// List returns a list of registered access providers.
func List() []string {
	mtx.Lock()
	defer mtx.Unlock()
	providers := make([]string, 0)
	for name := range registry {
		providers = append(providers, name)
	}
	return providers
}
