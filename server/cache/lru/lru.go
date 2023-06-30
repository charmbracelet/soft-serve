package lru

import (
	"context"

	"github.com/charmbracelet/soft-serve/server/cache"
	lru "github.com/hashicorp/golang-lru/v2"
)

func init() {
	cache.Register("lru", newCache)
}

// Cache is a memory cache that uses a LRU cache policy.
type Cache struct {
	cache   *lru.Cache[string, any]
	onEvict func(key string, value any)
	size    int
}

var _ cache.Cache = (*Cache)(nil)

// WithSize sets the cache size.
func WithSize(s int) cache.Option {
	return func(c cache.Cache) {
		ca := c.(*Cache)
		ca.size = s
	}
}

// WithEvictCallback sets the eviction callback.
func WithEvictCallback(cb func(key string, value any)) cache.Option {
	return func(c cache.Cache) {
		ca := c.(*Cache)
		ca.onEvict = cb
	}
}

// newCache returns a new Cache.
func newCache(_ context.Context, opts ...cache.Option) (cache.Cache, error) {
	c := &Cache{}
	for _, opt := range opts {
		opt(c)
	}

	if c.size <= 0 {
		c.size = 1
	}

	var err error
	c.cache, err = lru.NewWithEvict(c.size, c.onEvict)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// Delete implements cache.Cache.
func (c *Cache) Delete(_ context.Context, key string) {
	c.cache.Remove(key)
}

// Get implements cache.Cache.
func (c *Cache) Get(_ context.Context, key string) (value any, ok bool) {
	value, ok = c.cache.Get(key)
	return
}

// Keys implements cache.Cache.
func (c *Cache) Keys(_ context.Context) []string {
	return c.cache.Keys()
}

// Set implements cache.Cache.
func (c *Cache) Set(_ context.Context, key string, val any, _ ...cache.ItemOption) {
	c.cache.Add(key, val)
}

// Len implements cache.Cache.
func (c *Cache) Len(_ context.Context) int64 {
	return int64(c.cache.Len())
}

// Contains implements cache.Cache.
func (c *Cache) Contains(_ context.Context, key string) bool {
	return c.cache.Contains(key)
}
