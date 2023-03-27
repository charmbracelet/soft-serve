package server

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/utils"
	"github.com/dustin/go-humanize"
	"goji.io"
	"goji.io/pat"
	"goji.io/pattern"
)

// logWriter is a wrapper around http.ResponseWriter that allows us to capture
// the HTTP status code and bytes written to the response.
type logWriter struct {
	http.ResponseWriter
	code, bytes int
}

func (r *logWriter) Write(p []byte) (int, error) {
	written, err := r.ResponseWriter.Write(p)
	r.bytes += written
	return written, err
}

// Note this is generally only called when sending an HTTP error, so it's
// important to set the `code` value to 200 as a default
func (r *logWriter) WriteHeader(code int) {
	r.code = code
	r.ResponseWriter.WriteHeader(code)
}

func loggingMiddleware(next http.Handler) http.Handler {
	logger := logger.WithPrefix("server.http")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		writer := &logWriter{code: http.StatusOK, ResponseWriter: w}
		logger.Debug("request",
			"method", r.Method,
			"uri", r.RequestURI,
			"addr", r.RemoteAddr)
		next.ServeHTTP(writer, r)
		elapsed := time.Since(start)
		logger.Debug("response",
			"status", fmt.Sprintf("%d %s", writer.code, http.StatusText(writer.code)),
			"bytes", humanize.Bytes(uint64(writer.bytes)),
			"time", elapsed)
	})
}

// HTTPServer is an http server.
type HTTPServer struct {
	cfg        *config.Config
	server     *http.Server
	dirHandler http.Handler
}

func NewHTTPServer(cfg *config.Config) (*HTTPServer, error) {
	mux := goji.NewMux()
	s := &HTTPServer{
		cfg:        cfg,
		dirHandler: http.FileServer(http.Dir(filepath.Join(cfg.DataPath, "repos"))),
		server: &http.Server{
			Addr:              cfg.HTTP.ListenAddr,
			Handler:           mux,
			ReadHeaderTimeout: time.Second * 10,
			ReadTimeout:       time.Second * 10,
			WriteTimeout:      time.Second * 10,
			MaxHeaderBytes:    http.DefaultMaxHeaderBytes,
		},
	}

	mux.Use(loggingMiddleware)
	mux.HandleFunc(pat.Get("/:repo"), s.repoIndexHandler)
	mux.HandleFunc(pat.Get("/:repo/*"), s.dumbGitHandler)
	return s, nil
}

// Close closes the HTTP server.
func (s *HTTPServer) Close() error {
	return s.server.Close()
}

// ListenAndServe starts the HTTP server.
func (s *HTTPServer) ListenAndServe() error {
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the HTTP server.
func (s *HTTPServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

var repoIndexHTMLTpl = template.Must(template.New("index").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
	<meta http-equiv="Content-Type" content="text/html; charset=utf-8"/>
	<meta http-equiv="refresh" content="0; url=https://godoc.org/{{.ImportRoot}}/{{.Repo}}"
 	<meta name="go-import" content="{{.ImportRoot}}/{{.Repo}} git {{.Config.SSH.PublicURL}}/{{.Repo}}">
</head>
<body>
Redirecting to docs at <a href="https://godoc.org/{{.ImportRoot}}/{{.Repo}}">godoc.org/{{.ImportRoot}}/{{.Repo}}</a>...
</body>
</html>`))

func (s *HTTPServer) repoIndexHandler(w http.ResponseWriter, r *http.Request) {
	repo := pat.Param(r, "repo")
	repo = utils.SanitizeRepo(repo)

	// Only respond to go-get requests
	if r.URL.Query().Get("go-get") != "1" {
		http.NotFound(w, r)
		return
	}

	access := s.cfg.Backend.AccessLevel(repo, nil)
	if access < backend.ReadOnlyAccess {
		http.NotFound(w, r)
		return
	}

	importRoot, err := url.Parse(s.cfg.HTTP.PublicURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	if err := repoIndexHTMLTpl.Execute(w, struct {
		Repo       string
		Config     *config.Config
		ImportRoot string
	}{
		Repo:       repo,
		Config:     s.cfg,
		ImportRoot: importRoot.Host,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *HTTPServer) dumbGitHandler(w http.ResponseWriter, r *http.Request) {
	repo := pat.Param(r, "repo")
	repo = utils.SanitizeRepo(repo) + ".git"

	access := s.cfg.Backend.AccessLevel(repo, nil)
	if access < backend.ReadOnlyAccess || !s.cfg.Backend.AllowKeyless() {
		httpStatusError(w, http.StatusUnauthorized)
		return
	}

	path := pattern.Path(r.Context())
	stat, err := os.Stat(filepath.Join(s.cfg.DataPath, "repos", repo, path))
	// Restrict access to files
	if err != nil || stat.IsDir() {
		http.NotFound(w, r)
		return
	}

	// Don't allow access to non-git clients
	ua := r.Header.Get("User-Agent")
	if !strings.HasPrefix(strings.ToLower(ua), "git") {
		httpStatusError(w, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	r.URL.Path = fmt.Sprintf("/%s/%s", repo, path)
	s.dirHandler.ServeHTTP(w, r)
}

func httpStatusError(w http.ResponseWriter, status int) {
	http.Error(w, fmt.Sprintf("%d %s", status, http.StatusText(status)), status)
}
