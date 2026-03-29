// Package ratelimit provides a simple per-IP token-bucket rate limiter
// backed by golang.org/x/time/rate. It is intentionally minimal: one
// IPLimiter instance per server (SSH / HTTP / Git daemon) with a background
// goroutine that evicts stale entries.
package ratelimit

import (
	"net"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// IPLimiter tracks one token-bucket limiter per source IP address.
type IPLimiter struct {
	mu         sync.Mutex
	entries    map[string]*ipEntry
	r          rate.Limit
	burst      int
	ttl        time.Duration
	done       chan struct{}
	closeOnce  sync.Once
}

type ipEntry struct {
	lim      *rate.Limiter
	lastSeen time.Time
}

// New creates an IPLimiter that allows r tokens/second with the given burst
// per source IP. Idle entries are evicted after ttl.
func New(r rate.Limit, burst int, ttl time.Duration) *IPLimiter {
	il := &IPLimiter{
		entries: make(map[string]*ipEntry),
		r:       r,
		burst:   burst,
		ttl:     ttl,
		done:    make(chan struct{}),
	}
	go il.cleanup()
	return il
}

// Close stops the background cleanup goroutine.
func (il *IPLimiter) Close() {
	il.closeOnce.Do(func() { close(il.done) })
}

// Allow returns true if the source IP is within its rate limit.
// ip may be in "host:port" form; the port is stripped automatically.
func (il *IPLimiter) Allow(ip string) bool {
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	il.mu.Lock()
	e, ok := il.entries[ip]
	if !ok {
		e = &ipEntry{lim: rate.NewLimiter(il.r, il.burst)}
		il.entries[ip] = e
	}
	e.lastSeen = time.Now()
	allow := e.lim.Allow()
	il.mu.Unlock()
	return allow
}

func (il *IPLimiter) cleanup() {
	cleanupInterval := il.ttl / 5
	// cleanupInterval is clamped to a minimum of 1 minute regardless of TTL to avoid
	// excessive ticker overhead with very short TTLs.
	if cleanupInterval < time.Minute {
		cleanupInterval = time.Minute
	}
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			il.mu.Lock()
			for ip, e := range il.entries {
				if time.Since(e.lastSeen) > il.ttl {
					delete(il.entries, ip)
				}
			}
			il.mu.Unlock()
		case <-il.done:
			return
		}
	}
}
