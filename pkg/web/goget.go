package web

import (
	"net/http"
	"net/url"
	"path"
	"text/template"

	"charm.land/log/v2"
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
    <meta http-equiv="refresh" content="0; url=https://pkg.go.dev/{{ .ImportRoot }}/{{.Repo}}">
    <meta name="go-import" content="{{ .ImportRoot }}/{{ .Repo }} git {{ .CloneURL }}">
</head>
<body>
Redirecting to docs at <a href="https://pkg.go.dev/{{ .ImportRoot }}/{{ .Repo }}">pkg.go.dev/{{ .ImportRoot }}/{{ .Repo }}</a>...
</body>
</html>
`))

func GoGetHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cfg := config.FromContext(ctx)
	be := backend.FromContext(ctx)
	logger := log.FromContext(ctx).WithPrefix("http.go-get")

	repo := mux.Vars(r)["repo"]

	if r.URL.Query().Get("go-get") == "1" {
		repo := repo
		importRoot, err := url.Parse(cfg.HTTP.PublicURL)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

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

		sanitized := utils.SanitizeRepo(repo)
		cloneURL := cfg.HTTP.PublicURL + "/" + sanitized + ".git"
		if cfg.HTTP.StripGitSuffix {
			cloneURL = cfg.HTTP.PublicURL + "/" + sanitized
		}
		if err := repoIndexHTMLTpl.Execute(w, struct {
			Repo       string
			ImportRoot string
			CloneURL   string
		}{
			Repo:       sanitized,
			ImportRoot: importRoot.Host,
			CloneURL:   cloneURL,
		}); err != nil {
			logger.Error("failed to render go get template", "err", err)
			renderInternalServerError(w, r)
			return
		}

		goGetCounter.WithLabelValues(sanitized).Inc()
		return
	}

	renderNotFound(w, r)
}
