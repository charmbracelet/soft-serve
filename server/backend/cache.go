package backend

import lru "github.com/hashicorp/golang-lru/v2"

// TODO: implement a caching interface.
type cache struct {
	b     *Backend
	repos *lru.Cache[string, *repo]
}

func newCache(b *Backend, size int) *cache {
	if size <= 0 {
		size = 1
	}
	c := &cache{b: b}
	cache, _ := lru.New[string, *repo](size)
	c.repos = cache
	return c
}

func (c *cache) Get(repo string) (*repo, bool) {
	return c.repos.Get(repo)
}

func (c *cache) Set(repo string, r *repo) {
	c.repos.Add(repo, r)
}

func (c *cache) Delete(repo string) {
	c.repos.Remove(repo)
}

func (c *cache) Len() int {
	return c.repos.Len()
}
