package web

import (
	"context"
	"net/http"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// NewRouter returns a new HTTP router.
func NewRouter(ctx context.Context) http.Handler {
	router := mux.NewRouter()

	// Git routes
	GitController(ctx, router)

	router.PathPrefix("/").HandlerFunc(renderNotFound)

	// Context handler
	// Adds context to the request
	h := NewContextHandler(ctx)(router)
	h = handlers.CompressHandler(h)
	h = handlers.RecoveryHandler()(h)
	h = NewLoggingMiddleware(h)

	return h
}
