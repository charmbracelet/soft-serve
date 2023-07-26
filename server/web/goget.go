package web

import (
	"net/http"
	"net/url"
	"path"
	"text/template"

	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"goji.io/pattern"
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
    <meta name="go-import" content="{{ .ImportRoot }}/{{ .Repo }} git {{ .Config.HTTP.PublicURL }}/{{ .Repo }}">
</head>
<body>
Redirecting to docs at <a href="https://godoc.org/{{ .ImportRoot }}/{{ .Repo }}">godoc.org/{{ .ImportRoot }}/{{ .Repo }}</a>...
</body>
</html>
`))

// GoGetHandler handles go get requests.
type GoGetHandler struct{}

var _ http.Handler = (*GoGetHandler)(nil)

func (g GoGetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	repo := pattern.Path(r.Context())
	repo = utils.SanitizeRepo(repo)
	ctx := r.Context()
	cfg := config.FromContext(ctx)
	be := backend.FromContext(ctx)

	// Handle go get requests.
	//
	// Always return a 200 status code, even if the repo doesn't exist.
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
				return
			}

			repo = path.Dir(repo)
		}

		if err := repoIndexHTMLTpl.Execute(w, struct {
			Repo       string
			Config     *config.Config
			ImportRoot string
		}{
			Repo:       url.PathEscape(repo),
			Config:     cfg,
			ImportRoot: importRoot.Host,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		goGetCounter.WithLabelValues(repo).Inc()
		return
	}

	http.NotFound(w, r)
}
