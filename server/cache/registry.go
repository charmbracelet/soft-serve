package cache

import (
	"context"
	"fmt"
	"sync"
)

// Constructor is a function that returns a new cache.
type Constructor func(context.Context, ...Option) (Cache, error)

var (
	registry = map[string]Constructor{}
	mtx      sync.RWMutex

	// ErrCacheNotFound is returned when a cache is not found.
	ErrCacheNotFound = fmt.Errorf("cache not found")
)

// Register registers a cache.
func Register(name string, fn Constructor) {
	mtx.Lock()
	defer mtx.Unlock()

	registry[name] = fn
}

// New returns a new cache.
func New(name string, ctx context.Context, opts ...Option) (Cache, error) {
	mtx.RLock()
	fn, ok := registry[name]
	mtx.RUnlock()

	if !ok {
		return nil, ErrCacheNotFound
	}

	return fn(ctx, opts...)
}
