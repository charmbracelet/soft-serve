package backend

import (
	"context"
	"sync"
	"time"

	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/db"
)

// cachedBool is a simple time-based cache for a boolean value.
type cachedBool struct {
	mu        sync.Mutex
	val       bool
	expiresAt time.Time
}

func (c *cachedBool) get(ttl time.Duration, fetch func() (bool, error)) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if time.Now().Before(c.expiresAt) {
		return c.val, nil
	}
	v, err := fetch()
	if err != nil {
		return false, err
	}
	c.val = v
	c.expiresAt = time.Now().Add(ttl)
	return v, nil
}

const settingsCacheTTL = 30 * time.Second

var (
	allowKeylessCache cachedBool
	anonAccessCache   struct {
		mu        sync.Mutex
		val       access.AccessLevel
		expiresAt time.Time
	}
)

// AllowKeyless returns whether or not keyless access is allowed.
//
// It implements backend.Backend.
func (b *Backend) AllowKeyless(ctx context.Context) bool {
	val, err := allowKeylessCache.get(settingsCacheTTL, func() (bool, error) {
		var allow bool
		if err := b.db.TransactionContext(ctx, func(tx *db.Tx) error {
			var err error
			allow, err = b.store.GetAllowKeylessAccess(ctx, tx)
			return err
		}); err != nil {
			return false, err
		}
		return allow, nil
	})
	if err != nil {
		return false
	}
	return val
}

// SetAllowKeyless sets whether or not keyless access is allowed.
//
// It implements backend.Backend.
func (b *Backend) SetAllowKeyless(ctx context.Context, allow bool) error {
	if err := b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return b.store.SetAllowKeylessAccess(ctx, tx, allow)
	}); err != nil {
		return err
	}
	// Invalidate cache on write.
	allowKeylessCache.mu.Lock()
	allowKeylessCache.expiresAt = time.Time{}
	allowKeylessCache.mu.Unlock()
	return nil
}

// AnonAccess returns the level of anonymous access.
//
// It implements backend.Backend.
func (b *Backend) AnonAccess(ctx context.Context) access.AccessLevel {
	anonAccessCache.mu.Lock()
	defer anonAccessCache.mu.Unlock()
	if time.Now().Before(anonAccessCache.expiresAt) {
		return anonAccessCache.val
	}
	var level access.AccessLevel
	if err := b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		level, err = b.store.GetAnonAccess(ctx, tx)
		return err
	}); err != nil {
		return access.NoAccess
	}
	anonAccessCache.val = level
	anonAccessCache.expiresAt = time.Now().Add(settingsCacheTTL)
	return level
}

// SetAnonAccess sets the level of anonymous access.
//
// It implements backend.Backend.
func (b *Backend) SetAnonAccess(ctx context.Context, level access.AccessLevel) error {
	if err := b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return b.store.SetAnonAccess(ctx, tx, level)
	}); err != nil {
		return err
	}
	// Invalidate cache on write.
	anonAccessCache.mu.Lock()
	anonAccessCache.expiresAt = time.Time{}
	anonAccessCache.mu.Unlock()
	return nil
}
