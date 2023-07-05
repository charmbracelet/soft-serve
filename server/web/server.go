// Package server is the reusable server
package web

import (
	"context"
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/config"
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
	cfg := config.FromContext(ctx)
	be := backend.FromContext(ctx)
	logger := log.FromContext(ctx).WithPrefix("http")

	// Middlewares
	mux.Use(NewLoggingMiddleware(logger))

	// Git routes
	for _, service := range gitRoutes(ctx, logger) {
		mux.Handle(service, service)
	}

	// go-get handler
	mux.Handle(pat.Get("/*"), GoGetHandler{cfg, be})

	return mux
}
