package store

import (
	"context"
	"errors"
	"sync"

	"github.com/go-git/go-billy/v5"
)

// Constructor is a function that returns a new store.
type Constructor func(ctx context.Context, fs billy.Filesystem) (Store, error)

var (
	registry = map[string]Constructor{}
	mtx      sync.RWMutex

	// ErrStoreNotFound is returned when a store is not found.
	ErrStoreNotFound = errors.New("store not found")
)

// Register registers a store.
func Register(name string, fn Constructor) {
	mtx.Lock()
	defer mtx.Unlock()

	registry[name] = fn
}

// New returns a new store.
func New(ctx context.Context, fs billy.Filesystem, name string) (Store, error) {
	mtx.RLock()
	fn, ok := registry[name]
	mtx.RUnlock()

	if !ok {
		return nil, ErrStoreNotFound
	}

	return fn(ctx, fs)
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
