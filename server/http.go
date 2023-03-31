package server

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
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
	match func(*url.URL) *Match
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
type Matcher = func(*url.URL) *Match

var (
	getInfoRefs = func(u *url.URL) *Match {
		return matchSuffix(u.Path, "/info/refs")
	}

	getHead = func(u *url.URL) *Match {
		return matchSuffix(u.Path, "/HEAD")
	}

	getAlternates = func(u *url.URL) *Match {
		return matchSuffix(u.Path, "/objects/info/alternates")
	}

	getHTTPAlternates = func(u *url.URL) *Match {
		return matchSuffix(u.Path, "/objects/info/http-alternates")
	}

	getInfoPacks = func(u *url.URL) *Match {
		return matchSuffix(u.Path, "/objects/info/packs")
	}

	getInfoFileRegexp = regexp.MustCompile(".*?(/objects/info/[^/]*)$")
	getInfoFile       = func(u *url.URL) *Match {
		return findStringSubmatch(u.Path, getInfoFileRegexp)
	}

	getLooseObjectRegexp = regexp.MustCompile(".*?(/objects/[0-9a-f]{2}/[0-9a-f]{38})$")
	getLooseObject       = func(u *url.URL) *Match {
		return findStringSubmatch(u.Path, getLooseObjectRegexp)
	}

	getPackFileRegexp = regexp.MustCompile(".*?(/objects/pack/pack-[0-9a-f]{40}\\.pack)$")
	getPackFile       = func(u *url.URL) *Match {
		return findStringSubmatch(u.Path, getPackFileRegexp)
	}

	getIdxFileRegexp = regexp.MustCompile(".*?(/objects/pack/pack-[0-9a-f]{40}\\.idx)$")
	getIdxFile       = func(u *url.URL) *Match {
		return findStringSubmatch(u.Path, getIdxFileRegexp)
	}
)

type Match struct {
	RepoPath, FilePath string
}

func matchSuffix(path, suffix string) *Match {
	if !strings.HasSuffix(path, suffix) {
		return nil
	}
	repoPath := strings.Replace(path, suffix, "", 1)
	filePath := strings.Replace(path, repoPath+"/", "", 1)
	return &Match{repoPath, filePath}
}

func findStringSubmatch(path string, prefix *regexp.Regexp) *Match {
	m := prefix.FindStringSubmatch(path)
	if m == nil {
		return nil
	}
	suffix := m[1]
	repoPath := strings.Replace(path, suffix, "", 1)
	filePath := strings.Replace(path, repoPath+"/", "", 1)
	return &Match{repoPath, filePath}
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

func (s *HTTPServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	repo := pattern.Path(r.Context())
	repo = utils.SanitizeRepo(repo)
	if _, err := s.cfg.Backend.Repository(repo); err != nil {
		http.NotFound(w, r)
		return
	}

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
		return
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

func (s *HTTPServer) handleGit(w http.ResponseWriter, r *http.Request) {
	repo := pat.Param(r, "repo")
	repo = utils.SanitizeRepo(repo) + ".git"
	if _, err := s.cfg.Backend.Repository(repo); err != nil {
		logger.Debug("repository not found", "repo", repo, "err", err)
		http.NotFound(w, r)
		return
	}

	if !s.cfg.Backend.AllowKeyless() {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	access := s.cfg.Backend.AccessLevel(repo, nil)
	if access < backend.ReadOnlyAccess {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	file := pat.Param(r, "file")
	r.URL.Path = fmt.Sprintf("/%s/%s", repo, file)
	s.dirHandler.ServeHTTP(w, r)
}
