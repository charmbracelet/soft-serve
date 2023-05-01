package web

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/utils"
	"github.com/dustin/go-humanize"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"goji.io"
	"goji.io/pat"
	"goji.io/pattern"
)

var (
	gitHttpCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "soft_serve",
		Subsystem: "http",
		Name:      "git_fetch_pull_total",
		Help:      "The total number of git fetch/pull requests",
	}, []string{"repo", "file"})

	goGetCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "soft_serve",
		Subsystem: "http",
		Name:      "go_get_total",
		Help:      "The total number of go get requests",
	}, []string{"repo"})
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

func (s *HTTPServer) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		writer := &logWriter{code: http.StatusOK, ResponseWriter: w}
		s.logger.Debug("request",
			"method", r.Method,
			"uri", r.RequestURI,
			"addr", r.RemoteAddr)
		next.ServeHTTP(writer, r)
		elapsed := time.Since(start)
		s.logger.Debug("response",
			"status", fmt.Sprintf("%d %s", writer.code, http.StatusText(writer.code)),
			"bytes", humanize.Bytes(uint64(writer.bytes)),
			"time", elapsed)
	})
}

// HTTPServer is an http server.
type HTTPServer struct {
	ctx        context.Context
	cfg        *config.Config
	server     *http.Server
	dirHandler http.Handler
	logger     *log.Logger
}

func NewHTTPServer(ctx context.Context) (*HTTPServer, error) {
	cfg := config.FromContext(ctx)
	mux := goji.NewMux()
	s := &HTTPServer{
		ctx:        ctx,
		cfg:        cfg,
		logger:     log.FromContext(ctx).WithPrefix("http"),
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

	mux.Use(s.loggingMiddleware)
	for _, m := range []Matcher{
		getInfoRefs,
		getHead,
		getAlternates,
		getHTTPAlternates,
		getInfoPacks,
		getInfoFile,
		getLooseObject,
		getPackFile,
		getIdxFile,
	} {
		mux.HandleFunc(NewPattern(m), s.handleGit)
	}
	mux.HandleFunc(pat.Get("/*"), s.handleIndex)
	return s, nil
}

// Close closes the HTTP server.
func (s *HTTPServer) Close() error {
	return s.server.Close()
}

// ListenAndServe starts the HTTP server.
func (s *HTTPServer) ListenAndServe() error {
	if s.cfg.HTTP.TLSKeyPath != "" && s.cfg.HTTP.TLSCertPath != "" {
		return s.server.ListenAndServeTLS(s.cfg.HTTP.TLSCertPath, s.cfg.HTTP.TLSKeyPath)
	}
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the HTTP server.
func (s *HTTPServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// Pattern is a pattern for matching a URL.
// It matches against GET requests.
type Pattern struct {
	match func(*url.URL) *match
}

// NewPattern returns a new Pattern with the given matcher.
func NewPattern(m Matcher) *Pattern {
	return &Pattern{
		match: m,
	}
}

// Match is a match for a URL.
//
// It implements goji.Pattern.
func (p *Pattern) Match(r *http.Request) *http.Request {
	if r.Method != "GET" {
		return nil
	}

	if m := p.match(r.URL); m != nil {
		ctx := context.WithValue(r.Context(), pattern.Variable("repo"), m.RepoPath)
		ctx = context.WithValue(ctx, pattern.Variable("file"), m.FilePath)
		return r.WithContext(ctx)
	}
	return nil
}

// Matcher finds a match in a *url.URL.
type Matcher = func(*url.URL) *match

var (
	getInfoRefs = func(u *url.URL) *match {
		return matchSuffix(u.Path, "/info/refs")
	}

	getHead = func(u *url.URL) *match {
		return matchSuffix(u.Path, "/HEAD")
	}

	getAlternates = func(u *url.URL) *match {
		return matchSuffix(u.Path, "/objects/info/alternates")
	}

	getHTTPAlternates = func(u *url.URL) *match {
		return matchSuffix(u.Path, "/objects/info/http-alternates")
	}

	getInfoPacks = func(u *url.URL) *match {
		return matchSuffix(u.Path, "/objects/info/packs")
	}

	getInfoFileRegexp = regexp.MustCompile(".*?(/objects/info/[^/]*)$")
	getInfoFile       = func(u *url.URL) *match {
		return findStringSubmatch(u.Path, getInfoFileRegexp)
	}

	getLooseObjectRegexp = regexp.MustCompile(".*?(/objects/[0-9a-f]{2}/[0-9a-f]{38})$")
	getLooseObject       = func(u *url.URL) *match {
		return findStringSubmatch(u.Path, getLooseObjectRegexp)
	}

	getPackFileRegexp = regexp.MustCompile(`.*?(/objects/pack/pack-[0-9a-f]{40}\.pack)$`)
	getPackFile       = func(u *url.URL) *match {
		return findStringSubmatch(u.Path, getPackFileRegexp)
	}

	getIdxFileRegexp = regexp.MustCompile(`.*?(/objects/pack/pack-[0-9a-f]{40}\.idx)$`)
	getIdxFile       = func(u *url.URL) *match {
		return findStringSubmatch(u.Path, getIdxFileRegexp)
	}
)

// match represents a match for a URL.
type match struct {
	RepoPath, FilePath string
}

func matchSuffix(path, suffix string) *match {
	if !strings.HasSuffix(path, suffix) {
		return nil
	}
	repoPath := strings.Replace(path, suffix, "", 1)
	filePath := strings.Replace(path, repoPath+"/", "", 1)
	return &match{repoPath, filePath}
}

func findStringSubmatch(path string, prefix *regexp.Regexp) *match {
	m := prefix.FindStringSubmatch(path)
	if m == nil {
		return nil
	}
	suffix := m[1]
	repoPath := strings.Replace(path, suffix, "", 1)
	filePath := strings.Replace(path, repoPath+"/", "", 1)
	return &match{repoPath, filePath}
}

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
</html>`))

func (s *HTTPServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	repo := pattern.Path(r.Context())
	repo = utils.SanitizeRepo(repo)

	// Handle go get requests.
	//
	// Always return a 200 status code, even if the repo doesn't exist.
	//
	// https://golang.org/cmd/go/#hdr-Remote_import_paths
	// https://go.dev/ref/mod#vcs-branch
	if r.URL.Query().Get("go-get") == "1" {
		repo := repo
		importRoot, err := url.Parse(s.cfg.HTTP.PublicURL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// find the repo
		for {
			if _, err := s.cfg.Backend.Repository(repo); err == nil {
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
			Config:     s.cfg,
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

func (s *HTTPServer) handleGit(w http.ResponseWriter, r *http.Request) {
	repo := pat.Param(r, "repo")
	repo = utils.SanitizeRepo(repo) + ".git"
	if _, err := s.cfg.Backend.Repository(repo); err != nil {
		s.logger.Debug("repository not found", "repo", repo, "err", err)
		http.NotFound(w, r)
		return
	}

	if !s.cfg.Backend.AllowKeyless() {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	access := s.cfg.Backend.AccessLevel(repo, "")
	if access < backend.ReadOnlyAccess {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	file := pat.Param(r, "file")
	gitHttpCounter.WithLabelValues(repo, file).Inc()
	r.URL.Path = fmt.Sprintf("/%s/%s", repo, file)
	s.dirHandler.ServeHTTP(w, r)
}
