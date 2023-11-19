package web

import (
	"context"
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/store"
)

// NewContextHandler returns a new context middleware.
// This middleware adds the config, backend, and logger to the request context.
func NewContextHandler(ctx context.Context) func(http.Handler) http.Handler {
	cfg := config.FromContext(ctx)
	be := backend.FromContext(ctx)
	logger := log.FromContext(ctx).WithPrefix("http")
	dbx := db.FromContext(ctx)
	datastore := store.FromContext(ctx)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ctx = config.WithContext(ctx, cfg)
			ctx = backend.WithContext(ctx, be)
			ctx = log.WithContext(ctx, logger.With(
				"method", r.Method,
				"path", r.URL,
				"addr", r.RemoteAddr,
			))
			ctx = db.WithContext(ctx, dbx)
			ctx = store.WithContext(ctx, datastore)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}
