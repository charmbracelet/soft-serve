package web

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"charm.land/log/v2"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/ratelimit"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"golang.org/x/time/rate"
)

// realClientIP extracts the real client IP from the request.
// When trustProxyHeaders is true it uses the leftmost value of
// X-Forwarded-For; otherwise it falls back to RemoteAddr (with port stripped).
func realClientIP(r *http.Request, trustProxyHeaders bool) string {
	if trustProxyHeaders {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			// The leftmost IP is the original client.
			if idx := strings.IndexByte(xff, ','); idx != -1 {
				return strings.TrimSpace(xff[:idx])
			}
			return strings.TrimSpace(xff)
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func newRateLimitMiddleware(limiter *ratelimit.IPLimiter, trustProxyHeaders bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow(realClientIP(r, trustProxyHeaders)) {
				http.Error(w, "too many requests", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// securityHeadersMiddleware sets defensive HTTP response headers.
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'none'")
		next.ServeHTTP(w, r)
	})
}

// NewRouter returns a new HTTP router.
func NewRouter(ctx context.Context) http.Handler {
	logger := log.FromContext(ctx).WithPrefix("http")
	router := mux.NewRouter()

	// Health routes
	HealthController(ctx, router)

	// Git routes
	GitController(ctx, router)

	router.PathPrefix("/").HandlerFunc(renderNotFound)

	cfg := config.FromContext(ctx)
	httpLimiter := ratelimit.New(rate.Limit(100), 200, 10*time.Minute)

	// Context handler
	// Adds context to the request
	h := NewLoggingMiddleware(router, logger)
	h = NewContextHandler(ctx)(h)
	h = gitSuffixMiddleware(cfg)(h)
	h = handlers.CompressHandler(h)
	h = handlers.RecoveryHandler()(h)
	h = securityHeadersMiddleware(h)
	h = newRateLimitMiddleware(httpLimiter, cfg.HTTP.TrustProxyHeaders)(h)

	h = handlers.CORS(handlers.AllowedHeaders(cfg.HTTP.CORS.AllowedHeaders),
		handlers.AllowedOrigins(cfg.HTTP.CORS.AllowedOrigins),
		handlers.AllowedMethods(cfg.HTTP.CORS.AllowedMethods),
	)(h)

	return h
}
