package web

import (
	"net/http"
	"net/url"
	"path"
	"text/template"

	"github.com/charmbracelet/log/v2"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/utils"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var goGetCounter = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "soft_serve",
	Subsystem: "http",
	Name:      "go_get_total",
	Help:      "The total number of go get requests",
}, []string{"repo"})

var repoIndexHTMLTpl = template.Must(template.New("index").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta http-equiv="Content-Type" content="text/html; charset=utf-8"/>
    <meta http-equiv="refresh" content="0; url=https://godoc.org/{{ .ImportRoot }}/{{.Repo}}">
    <meta name="go-import" content="{{ .ImportRoot }}/{{ .Repo }} git {{ .Config.HTTP.PublicURL }}/{{ .Repo }}.git">
</head>
<body>
Redirecting to docs at <a href="https://godoc.org/{{ .ImportRoot }}/{{ .Repo }}">godoc.org/{{ .ImportRoot }}/{{ .Repo }}</a>...
</body>
</html>
`))

// GoGetHandler handles go get requests.
func GoGetHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cfg := config.FromContext(ctx)
	be := backend.FromContext(ctx)
	logger := log.FromContext(ctx)
	repo := mux.Vars(r)["repo"]

	// Handle go get requests.
	//
	// Always return a 200 status code, even if the repo path doesn't exist.
	// It will try to find the repo by walking up the path until it finds one.
	// If it can't find one, it will return a 404.
	//
	// https://golang.org/cmd/go/#hdr-Remote_import_paths
	// https://go.dev/ref/mod#vcs-branch
	if r.URL.Query().Get("go-get") == "1" {
		repo := repo
		importRoot, err := url.Parse(cfg.HTTP.PublicURL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// find the repo
		for {
			if _, err := be.Repository(ctx, repo); err == nil {
				break
			}

			if repo == "" || repo == "." || repo == "/" {
				renderNotFound(w, r)
				return
			}

			repo = path.Dir(repo)
		}

		if err := repoIndexHTMLTpl.Execute(w, struct {
			Repo       string
			Config     *config.Config
			ImportRoot string
		}{
			Repo:       utils.SanitizeRepo(repo),
			Config:     cfg,
			ImportRoot: importRoot.Host,
		}); err != nil {
			logger.Error("failed to render go get template", "err", err)
			renderInternalServerError(w, r)
			return
		}

		goGetCounter.WithLabelValues(repo).Inc()
		return
	}

	renderNotFound(w, r)
}
