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
//
// WARNING: Only enable trustProxyHeaders when the server is behind a single
// trusted reverse proxy that overwrites (not appends to) X-Forwarded-For.
// If clients or untrusted proxies can set this header, the leftmost IP can
// be spoofed, defeating rate-limiting and audit-logging based on client IP.
func realClientIP(r *http.Request, trustProxyHeaders bool) string {
	if trustProxyHeaders {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			// The leftmost IP is the original client.
			var candidate string
			if idx := strings.IndexByte(xff, ','); idx != -1 {
				candidate = strings.TrimSpace(xff[:idx])
			} else {
				candidate = strings.TrimSpace(xff)
			}
			// Validate that the candidate is a real IP address before using it
			// as the rate-limit key. An attacker who can set X-Forwarded-For
			// could otherwise bypass per-IP limiting with arbitrary strings.
			if net.ParseIP(candidate) != nil {
				return candidate
			}
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

// securityHeadersMiddleware returns middleware that sets defensive HTTP response headers.
// When TLS is configured (both TLSKeyPath and TLSCertPath are set) it also adds an HSTS header.
func securityHeadersMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Content-Security-Policy", "default-src 'none'")
			// Only add HSTS when serving over TLS
			if cfg.HTTP.TLSKeyPath != "" && cfg.HTTP.TLSCertPath != "" {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}
			next.ServeHTTP(w, r)
		})
	}
}

// NewRouter returns a new HTTP router and the rate limiter that must be closed
// when the server shuts down.
func NewRouter(ctx context.Context) (http.Handler, *ratelimit.IPLimiter) {
	logger := log.FromContext(ctx).WithPrefix("http")
	router := mux.NewRouter()

	// Health routes
	HealthController(ctx, router)

	// Git routes
	GitController(ctx, router)

	router.PathPrefix("/").HandlerFunc(renderNotFound)

	cfg := config.FromContext(ctx)
	// Only create the rate limiter (and its background cleanup goroutine) when
	// rate limiting is actually enabled. Callers handle a nil limiter.
	var httpLimiter *ratelimit.IPLimiter
	if cfg.HTTP.RateLimit > 0 {
		httpLimiter = ratelimit.New(rate.Limit(cfg.HTTP.RateLimit), cfg.HTTP.RateBurst, 10*time.Minute)
	}

	// Context handler
	// Adds context to the request
	h := NewLoggingMiddleware(router, logger)
	h = NewContextHandler(ctx)(h)
	h = gitSuffixMiddleware(cfg)(h)
	h = handlers.CompressHandler(h)
	h = handlers.RecoveryHandler()(h)
	h = securityHeadersMiddleware(cfg)(h)
	// Note: CORS middleware is applied before the rate limiter so that OPTIONS
	// preflights receive CORS headers even when rate-limited (the browser needs
	// those headers to understand the 429 response). All request methods,
	// including OPTIONS, consume rate-limit tokens.
	//
	// AllowCredentials is explicitly NOT set (defaults to false). Wildcard
	// AllowedOrigins ("*") is incompatible with credential-carrying requests;
	// browsers reject Access-Control-Allow-Credentials: true when the origin
	// is "*". Operators who need credentialed cross-origin access must specify
	// explicit origins in the CORS config.
	for _, origin := range cfg.HTTP.CORS.AllowedOrigins {
		if origin == "*" {
			for _, header := range cfg.HTTP.CORS.AllowedHeaders {
				if strings.EqualFold(header, "Authorization") {
					logger.Warn("CORS: wildcard AllowedOrigins with Authorization in AllowedHeaders — " +
						"any origin can trigger credentialless requests; consider restricting AllowedOrigins")
					break
				}
			}
			break
		}
	}
	h = handlers.CORS(handlers.AllowedHeaders(cfg.HTTP.CORS.AllowedHeaders),
		handlers.AllowedOrigins(cfg.HTTP.CORS.AllowedOrigins),
		handlers.AllowedMethods(cfg.HTTP.CORS.AllowedMethods),
	)(h)
	if cfg.HTTP.RateLimit > 0 {
		h = newRateLimitMiddleware(httpLimiter, cfg.HTTP.TrustProxyHeaders)(h)
	}

	return h, httpLimiter
}
