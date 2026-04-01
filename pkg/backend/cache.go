package backend

import (
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

const (
	defaultCacheSize = 1000
	defaultCacheTTL  = 5 * time.Minute
)

type cacheEntry struct {
	repo      *repo
	expiresAt time.Time
}

type cache struct {
	b     *Backend
	repos *lru.Cache[string, *cacheEntry]
	ttl   time.Duration
}

func newCache(b *Backend, size int) *cache {
	if size <= 0 {
		size = 1
	}
	c := &cache{b: b, ttl: defaultCacheTTL}
	cache, err := lru.New[string, *cacheEntry](size)
	if err != nil {
		b.logger.Error("failed to create LRU cache, using size 1", "err", err)
		cache, _ = lru.New[string, *cacheEntry](1)
	}
	c.repos = cache
	return c
}

func (c *cache) Get(repo string) (*repo, bool) {
	if entry, ok := c.repos.Get(repo); ok && entry != nil {
		if time.Now().Before(entry.expiresAt) {
			return entry.repo, true
		}
		c.repos.Remove(repo)
	}
	return nil, false
}

func (c *cache) Set(repo string, r *repo) {
	c.repos.Add(repo, &cacheEntry{
		repo:      r,
		expiresAt: time.Now().Add(c.ttl),
	})
}

func (c *cache) Delete(repo string) {
	c.repos.Remove(repo)
}
