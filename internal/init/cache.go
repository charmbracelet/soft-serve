package init

import (
	"github.com/charmbracelet/soft-serve/server/cache"
	"github.com/charmbracelet/soft-serve/server/cache/lru"
	"github.com/charmbracelet/soft-serve/server/cache/noop"
	"github.com/charmbracelet/soft-serve/server/cache/redis"
)

func init() {
	cache.Register("lru", lru.NewCache)
	cache.Register("noop", noop.NewCache)
	cache.Register("redis", redis.NewCache)
}
