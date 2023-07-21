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
func NewRouter(ctx context.Context) *goji.Mux {
	mux := goji.NewMux()

	// Middlewares
	mux.Use(NewContextMiddleware(ctx))
	mux.Use(NewLoggingMiddleware)

	// Git routes
	for _, service := range gitRoutes {
		mux.Handle(service, withAccess(service.handler))
	}

	// go-get handler
	mux.Handle(pat.Get("/*"), GoGetHandler{})

	return mux
}
