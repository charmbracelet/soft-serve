package auth

import (
	"context"
	"errors"
	"sync"
)

var (
	registry = map[string]Constructor{}
	mtx      sync.RWMutex

	// ErrNotFound is returned when a store is not found.
	ErrNotFound = errors.New("auth store not found")
)

// Constructor is a function that returns a new store.
type Constructor func(ctx context.Context) (Auth, error)

// Register registers a store.
func Register(name string, fn Constructor) {
	mtx.Lock()
	defer mtx.Unlock()

	registry[name] = fn
}

// New returns a new store.
func New(ctx context.Context, name string) (Auth, error) {
	mtx.RLock()
	fn, ok := registry[name]
	mtx.RUnlock()

	if !ok {
		return nil, ErrNotFound
	}

	return fn(ctx)
}

// List returns a list of registered stores.
func List() []string {
	mtx.Lock()
	defer mtx.Unlock()
	stores := make([]string, 0)
	for name := range registry {
		stores = append(stores, name)
	}
	return stores
}
