package cache

import (
	"context"
)

// ItemOption is an option for setting cache items.
type ItemOption func(Item)

// Item is an interface that represents a cache item.
type Item interface {
	item()
}

// Option is an option for creating new cache.
type Option func(Cache)

// Cache is a caching interface.
type Cache interface {
	Get(ctx context.Context, key string) (value any, ok bool)
	Set(ctx context.Context, key string, val any, opts ...ItemOption)
	Keys(ctx context.Context) []string
	Len(ctx context.Context) int64
	Contains(ctx context.Context, key string) bool
	Delete(ctx context.Context, key string)
}
