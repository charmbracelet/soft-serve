package web

import (
	"context"
	"net/http"

	"github.com/charmbracelet/log/v2"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// NewRouter returns a new HTTP router.
func NewRouter(ctx context.Context) http.Handler {
	logger := log.FromContext(ctx).WithPrefix("http")
	router := mux.NewRouter()

	// Health routes
	HealthController(ctx, router)

	// Git routes
	GitController(ctx, router)

	router.PathPrefix("/").HandlerFunc(renderNotFound)

	// Context handler
	// Adds context to the request
	h := NewLoggingMiddleware(router, logger)
	h = NewContextHandler(ctx)(h)
	h = handlers.CompressHandler(h)
	h = handlers.RecoveryHandler()(h)

	return h
}
