package web

import (
	"context"
	"net/http"

	"github.com/charmbracelet/log/v2"
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
	logger := log.FromContext(ctx)
	db := db.FromContext(ctx)

	if err := db.PingContext(ctx); err != nil {
		logger.Error("error getting db readiness", "err", err)
		renderStatus(http.StatusServiceUnavailable)(w, nil)
		return
	}

	renderStatus(http.StatusOK)(w, nil)
}
