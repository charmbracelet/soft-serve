package noop

import (
	"context"

	"github.com/charmbracelet/soft-serve/server/cache"
)

func init() {
	cache.Register("noop", newCache)
}

type noopCache struct{}

// newCache returns a new Cache.
func newCache(_ context.Context, _ ...cache.Option) (cache.Cache, error) {
	return &noopCache{}, nil
}

// Contains implements Cache.
func (*noopCache) Contains(_ context.Context, _ string) bool {
	return false
}

// Delete implements Cache.
func (*noopCache) Delete(_ context.Context, _ string) {}

// Get implements Cache.
func (*noopCache) Get(_ context.Context, _ string) (any, bool) {
	return nil, false
}

// Keys implements Cache.
func (*noopCache) Keys(_ context.Context) []string {
	return []string{}
}

// Len implements Cache.
func (*noopCache) Len(_ context.Context) int64 {
	return -1
}

// Set implements Cache.
func (*noopCache) Set(_ context.Context, _ string, _ any, _ ...cache.ItemOption) {}

var _ cache.Cache = &noopCache{}
