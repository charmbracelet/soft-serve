package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/soft-serve/config"
	appCfg "github.com/charmbracelet/soft-serve/internal/config"
	"github.com/charmbracelet/wish/git"
	"goji.io"
	"goji.io/pat"
	"goji.io/pattern"
)

type HTTPServer struct {
	server     *http.Server
	gitHandler http.Handler
	cfg        *config.Config
	ac         *appCfg.Config
}

func NewHTTPServer(cfg *config.Config, ac *appCfg.Config) *HTTPServer {
	h := goji.NewMux()
	s := &HTTPServer{
		cfg:        cfg,
		ac:         ac,
		gitHandler: http.FileServer(http.Dir(cfg.RepoPath)),
		server: &http.Server{
			Addr:      fmt.Sprintf(":%d", cfg.HTTPPort),
			Handler:   h,
			TLSConfig: cfg.TLSConfig,
		},
	}
	h.HandleFunc(pat.Get("/:repo"), s.handleGit)
	h.HandleFunc(pat.Get("/:repo/*"), s.handleGit)
	return s
}

func (s *HTTPServer) Start() error {
	if s.cfg.TLSConfig != nil {
		return s.server.ListenAndServeTLS("", "")
	}
	return s.server.ListenAndServe()
}

func (s *HTTPServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *HTTPServer) handleGit(w http.ResponseWriter, r *http.Request) {
	ua := r.Header.Get("User-Agent")
	repo := pat.Param(r, "repo")
	access := s.ac.AuthRepo(repo, nil)
	path := pattern.Path(r.Context())
	stat, err := os.Stat(filepath.Join(s.cfg.RepoPath, repo, path))
	// Restrict access to files
	if err != nil || stat.IsDir() {
		http.NotFound(w, r)
		return
	}
	if !strings.HasPrefix(strings.ToLower(ua), "git") {
		http.Error(w, fmt.Sprintf("%d Bad Request", http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	if access < git.ReadOnlyAccess || !s.ac.AllowKeyless {
		http.Error(w, fmt.Sprintf("%d Unauthorized", http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	r.URL.Path = fmt.Sprintf("/%s/%s", repo, path)
	s.gitHandler.ServeHTTP(w, r)
}
