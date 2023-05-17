package redis

import (
	"context"
	"time"

	"github.com/charmbracelet/soft-serve/server/cache"
	"github.com/redis/go-redis/v9"
)

// Cache is a Redis cache.
type Cache struct {
	client *redis.Client
}

// NewCache returns a new Redis cache.
// It converts non-string types to JSON before storing/retrieving them.
func NewCache(ctx context.Context, _ ...cache.Option) (cache.Cache, error) {
	cfg, err := NewConfig("")
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Username: cfg.Username,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	return &Cache{
		client: client,
	}, client.Ping(ctx).Err()
}

type option struct {
	cache.Item
	ttl time.Duration
}

func (*option) item() {}

// WithTTL sets the TTL for the cache item.
func WithTTL(ttl time.Duration) cache.ItemOption {
	return func(io cache.Item) {
		i := io.(*option)
		i.ttl = ttl
	}
}

// Contains implements cache.Cache.
func (r *Cache) Contains(ctx context.Context, key string) bool {
	return r.client.Exists(ctx, key).Val() == 1
}

// Delete implements cache.Cache.
func (r *Cache) Delete(ctx context.Context, key string) {
	r.client.Del(ctx, key)
}

// Get implements cache.Cache.
func (r *Cache) Get(ctx context.Context, key string) (value any, ok bool) {
	val := r.client.Get(ctx, key)
	if val.Err() != nil {
		return nil, false
	}

	return val.Val(), true
}

// Keys implements cache.Cache.
func (r *Cache) Keys(ctx context.Context) []string {
	return r.client.Keys(ctx, "*").Val()
}

// Len implements cache.Cache.
func (r *Cache) Len(ctx context.Context) int64 {
	return r.client.DBSize(ctx).Val()
}

// Set implements cache.Cache.
func (r *Cache) Set(ctx context.Context, key string, val any, opts ...cache.ItemOption) {
	var opt option
	for _, o := range opts {
		o(&opt)
	}
	r.client.Set(ctx, key, val, opt.ttl)
}
