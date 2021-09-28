package git

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"text/template"

	"goji.io"
	"goji.io/pat"
	"goji.io/pattern"
)

const (
	// `curl https://golang.org/x/tools`
	repoMetaTpl = `
<!DOCTYPE html>
<html>
<head>
<meta http-equiv="Content-Type" content="text/html; charset=utf-8"/>
<meta name="go-import" content="{{.Prefix}} {{.Vcs}} {{.ImportUrl}}">
<meta http-equiv="refresh" content="0; url=https://pkg.go.dev/{{.Prefix}}{{.Suffix}}">
</head>
<body>
<a href="https://pkg.go.dev/{{.Prefix}}{{.Suffix}}">Redirecting to documentation...</a>
</body>
`
)

type HTTPServer struct {
	server *http.Server
	path   string
}

type repoMeta struct {
	Prefix    string
	Suffix    string
	ImportUrl string
	SourceUrl string
	Vcs       string
}

// NewHTTPServer creates a new HTTP dumb server
func NewHTTPServer(repoDir string) (*HTTPServer, error) {
	mux := goji.NewMux()
	s := &HTTPServer{
		server: &http.Server{
			Addr:    ":23232",
			Handler: mux,
			// TLSConfig: ,
		},
		path: repoDir,
	}
	domain := "beta.charm.sh"
	mux.HandleFunc(pat.Get("/go/:repo"), handleGo(repoDir, domain))
	mux.HandleFunc(pat.Get("/go/:repo/*"), handleGo(repoDir, domain))
	mux.HandleFunc(pat.Get("/git/:repo/*"), handleGit(repoDir))
	return s, nil
}

func (s *HTTPServer) Start() error {
	log.Printf("Starting a dumb Git HTTP server on %s...", s.server.Addr)
	return s.server.ListenAndServe()
}

func handleGo(reposPath, domain string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		repo := pat.Param(r, "repo")
		path := pattern.Path(r.Context())
		log.Println(path)
		info, err := os.Stat(filepath.Join(reposPath, repo))
		if err != nil || !info.IsDir() {
			http.NotFound(w, r)
			return
		}
		tmpl := template.Must(template.New("repoMeta").Parse(repoMetaTpl))
		// Render template
		err = tmpl.Execute(w, repoMeta{
			Prefix:    fmt.Sprintf("%s/%s", domain, repo),
			Suffix:    path,
			ImportUrl: fmt.Sprintf("ssh://%s/%s", domain, repo),
			SourceUrl: fmt.Sprintf("https://%s/%s", domain, repo),
			Vcs:       "git",
		})
		if err != nil {
			log.Print(err)
		}
	}
}

func handleGit(repoDir string) func(http.ResponseWriter, *http.Request) {
	fs := http.FileServer(http.Dir(repoDir))
	return func(w http.ResponseWriter, r *http.Request) {
		repo := pat.Param(r, "repo")
		path := pattern.Path(r.Context())
		stat, err := os.Stat(filepath.Join(repoDir, repo, path))
		// Restrict access to directories
		if err != nil || stat.IsDir() {
			http.NotFound(w, r)
			return
		}
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		r.URL.Path = fmt.Sprintf("/%s/%s", repo, path)
		fs.ServeHTTP(w, r)
	}
}
