package web

import (
	"context"
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/charmbracelet/soft-serve/pkg/config"
)

// NewRouter returns a new HTTP router.
func NewRouter(ctx context.Context) http.Handler {
	logger := log.FromContext(ctx).WithPrefix("http")
	router := mux.NewRouter()

	// Git routes
	GitController(ctx, router)

	router.PathPrefix("/").HandlerFunc(renderNotFound)

	// Context handler
	// Adds context to the request
	h := NewLoggingMiddleware(router, logger)
	h = NewContextHandler(ctx)(h)
	h = handlers.CompressHandler(h)
	h = handlers.RecoveryHandler()(h)

	cfg := config.FromContext(ctx)

	CORSHeaders := []string{"Accept", "Accept-Language", "Content-Language", "Origin"}

	if len(cfg.HTTP.CORS.AllowedHeaders) != 0 {
		CORSHeaders = cfg.HTTP.CORS.AllowedHeaders
	}

	CORSOrigins := []string{}

	if len(cfg.HTTP.CORS.AllowedOrigins) != 0 {
		CORSOrigins = cfg.HTTP.CORS.AllowedOrigins
	}

	CORSMethods := []string{http.MethodGet, http.MethodHead, http.MethodPost}

	if len(cfg.HTTP.CORS.AllowedMethods) != 0 {
		CORSMethods = cfg.HTTP.CORS.AllowedMethods
	}

	h = handlers.CORS(handlers.AllowedHeaders(CORSHeaders),handlers.AllowedOrigins(CORSOrigins),handlers.AllowedMethods(CORSMethods))(h)

	return h
}
