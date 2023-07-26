package web

import (
	"context"
	"net/http"

	"goji.io"
	"goji.io/pat"
)

// Route is an interface for a route.
type Route interface {
	http.Handler
	goji.Pattern
}

// NewRouter returns a new HTTP router.
// TODO: use gorilla/mux and friends
func NewRouter(ctx context.Context) http.Handler {
	mux := goji.NewMux()

	// Git routes
	for _, service := range gitRoutes {
		mux.Handle(service, withAccess(service))
	}

	// go-get handler
	mux.Handle(pat.Get("/*"), GoGetHandler{})

	// Middlewares
	mux.Use(NewLoggingMiddleware)

	// Context handler
	// Adds context to the request
	ctxHandler := NewContextHandler(ctx)

	return ctxHandler(mux)
}
