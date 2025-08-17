package web

import (
	"context"
	"fmt"
	"net/http"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/gorilla/mux"
)

// HealthController registers the health check routes for the web server.
func HealthController(_ context.Context, r *mux.Router) {
	r.HandleFunc("/livez", getLiveness)
	r.HandleFunc("/readyz", getReadiness)
}

func getLiveness(w http.ResponseWriter, _ *http.Request) {
	renderStatus(http.StatusOK)(w, nil)
}

func getReadiness(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	db := db.FromContext(ctx)

	errs := make([]error, 0)
	err := db.PingContext(ctx)
	if err != nil {
		errs = append(errs, fmt.Errorf("readiness check failed: %w", err))
	}

	if len(errs) > 0 {
		renderStatus(http.StatusServiceUnavailable)(w, nil)
		return
	}

	renderStatus(http.StatusOK)(w, nil)
}
