package web

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	gitb "github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/git"
	"github.com/charmbracelet/soft-serve/server/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"goji.io/pat"
	"goji.io/pattern"
)

// GitRoute is a route for git services.
type GitRoute struct {
	method  string
	pattern *regexp.Regexp
	handler http.HandlerFunc

	cfg    *config.Config
	be     backend.Backend
	logger *log.Logger
}

var _ Route = GitRoute{}

// Match implements goji.Pattern.
func (g GitRoute) Match(r *http.Request) *http.Request {
	if g.method != r.Method {
		return nil
	}

	re := g.pattern
	ctx := r.Context()
	if m := re.FindStringSubmatch(r.URL.Path); m != nil {
		file := strings.Replace(r.URL.Path, m[1]+"/", "", 1)
		repo := utils.SanitizeRepo(m[1]) + ".git"

		var service git.Service
		switch {
		case strings.HasSuffix(r.URL.Path, git.UploadPackService.String()):
			service = git.UploadPackService
		case strings.HasSuffix(r.URL.Path, git.ReceivePackService.String()):
			service = git.ReceivePackService
		}

		ctx = context.WithValue(ctx, pattern.Variable("service"), service.String())
		ctx = context.WithValue(ctx, pattern.Variable("dir"), filepath.Join(g.cfg.DataPath, "repos", repo))
		ctx = context.WithValue(ctx, pattern.Variable("repo"), repo)
		ctx = context.WithValue(ctx, pattern.Variable("file"), file)

		if g.cfg != nil {
			ctx = config.WithContext(ctx, g.cfg)
		}

		if g.be != nil {
			ctx = backend.WithContext(ctx, g.be.WithContext(ctx))
		}

		if g.logger != nil {
			ctx = log.WithContext(ctx, g.logger)
		}

		return r.WithContext(ctx)
	}

	return nil
}

// ServeHTTP implements http.Handler.
func (g GitRoute) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.handler(w, r)
}

var (
	gitHttpReceiveCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "soft_serve",
		Subsystem: "http",
		Name:      "git_receive_pack_total",
		Help:      "The total number of git push requests",
	}, []string{"repo"})

	gitHttpUploadCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "soft_serve",
		Subsystem: "http",
		Name:      "git_upload_pack_total",
		Help:      "The total number of git fetch/pull requests",
	}, []string{"repo", "file"})
)

func gitRoutes(ctx context.Context, logger *log.Logger) []Route {
	routes := make([]Route, 0)
	cfg := config.FromContext(ctx)
	be := backend.FromContext(ctx)

	// Git services
	// These routes don't handle authentication/authorization.
	// This is handled through wrapping the handlers for each route.
	// See below (withAccess).
	// TODO: add lfs support
	for _, route := range []GitRoute{
		{
			pattern: regexp.MustCompile("(.*?)/git-upload-pack$"),
			method:  http.MethodPost,
			handler: serviceRpc,
		},
		{
			pattern: regexp.MustCompile("(.*?)/git-receive-pack$"),
			method:  http.MethodPost,
			handler: serviceRpc,
		},
		{
			pattern: regexp.MustCompile("(.*?)/info/refs$"),
			method:  http.MethodGet,
			handler: getInfoRefs,
		},
		{
			pattern: regexp.MustCompile("(.*?)/HEAD$"),
			method:  http.MethodGet,
			handler: getTextFile,
		},
		{
			pattern: regexp.MustCompile("(.*?)/objects/info/alternates$"),
			method:  http.MethodGet,
			handler: getTextFile,
		},
		{
			pattern: regexp.MustCompile("(.*?)/objects/info/http-alternates$"),
			method:  http.MethodGet,
			handler: getTextFile,
		},
		{
			pattern: regexp.MustCompile("(.*?)/objects/info/packs$"),
			method:  http.MethodGet,
			handler: getInfoPacks,
		},
		{
			pattern: regexp.MustCompile("(.*?)/objects/info/[^/]*$"),
			method:  http.MethodGet,
			handler: getTextFile,
		},
		{
			pattern: regexp.MustCompile("(.*?)/objects/[0-9a-f]{2}/[0-9a-f]{38}$"),
			method:  http.MethodGet,
			handler: getLooseObject,
		},
		{
			pattern: regexp.MustCompile("(.*?)/objects/pack/pack-[0-9a-f]{40}\\.pack$"),
			method:  http.MethodGet,
			handler: getPackFile,
		},
		{
			pattern: regexp.MustCompile("(.*?)/objects/pack/pack-[0-9a-f]{40}\\.idx$"),
			method:  http.MethodGet,
			handler: getIdxFile,
		},
	} {
		route.cfg = cfg
		route.be = be
		route.logger = logger
		route.handler = withAccess(route.handler)
		routes = append(routes, route)
	}

	return routes
}

// withAccess handles auth.
func withAccess(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		be := backend.FromContext(ctx)
		logger := log.FromContext(ctx)

		if !be.AllowKeyless() {
			renderForbidden(w)
			return
		}

		repo := pat.Param(r, "repo")
		service := git.Service(pat.Param(r, "service"))
		access := be.AccessLevel(repo, "")

		switch service {
		case git.ReceivePackService:
			if access < backend.ReadWriteAccess {
				renderUnauthorized(w)
				return
			}

			// Create the repo if it doesn't exist.
			if _, err := be.Repository(repo); err != nil {
				if _, err := be.CreateRepository(repo, backend.RepositoryOptions{}); err != nil {
					logger.Error("failed to create repository", "repo", repo, "err", err)
					renderInternalServerError(w)
					return
				}
			}
		default:
			if access < backend.ReadOnlyAccess {
				renderUnauthorized(w)
				return
			}
		}

		fn(w, r)
	}
}

func serviceRpc(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx)
	service, dir, repo := git.Service(pat.Param(r, "service")), pat.Param(r, "dir"), pat.Param(r, "repo")

	if !isSmart(r, service) {
		renderForbidden(w)
		return
	}

	if service == git.ReceivePackService {
		gitHttpReceiveCounter.WithLabelValues(repo)
	}

	w.Header().Set("Content-Type", fmt.Sprintf("application/x-%s-result", service))
	w.Header().Set("Connection", "Keep-Alive")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)

	version := r.Header.Get("Git-Protocol")

	cmd := git.ServiceCommand{
		Stdin:  r.Body,
		Stdout: w,
		Dir:    dir,
		Args:   []string{"--stateless-rpc"},
	}

	if len(version) != 0 {
		cmd.Env = append(cmd.Env, fmt.Sprintf("GIT_PROTOCOL=%s", version))
	}

	// Handle gzip encoding
	cmd.StdinHandler = func(in io.Reader, stdin io.WriteCloser) (err error) {
		// We know that `in` is an `io.ReadCloser` because it's `r.Body`.
		reader := in.(io.ReadCloser)
		defer reader.Close() // nolint: errcheck
		switch r.Header.Get("Content-Encoding") {
		case "gzip":
			reader, err = gzip.NewReader(reader)
			if err != nil {
				return err
			}
			defer reader.Close() // nolint: errcheck
		}

		_, err = io.Copy(stdin, reader)
		return err
	}

	// Handle buffered output
	// Useful when using proxies
	cmd.StdoutHandler = func(out io.Writer, stdout io.ReadCloser) error {
		// We know that `out` is an `http.ResponseWriter`.
		flusher, ok := out.(http.Flusher)
		if !ok {
			return fmt.Errorf("expected http.ResponseWriter to be an http.Flusher, got %T", out)
		}

		p := make([]byte, 1024)
		for {
			nRead, err := stdout.Read(p)
			if err == io.EOF {
				break
			}
			nWrite, err := out.Write(p[:nRead])
			if err != nil {
				return err
			}
			if nRead != nWrite {
				return fmt.Errorf("failed to write data: %d read, %d written", nRead, nWrite)
			}
			flusher.Flush()
		}

		return nil
	}

	if err := service.Handler(ctx, cmd); err != nil {
		logger.Errorf("error executing service: %s", err)
	}
}

func getInfoRefs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx)
	dir, repo, file := pat.Param(r, "dir"), pat.Param(r, "repo"), pat.Param(r, "file")
	service := getServiceType(r)
	version := r.Header.Get("Git-Protocol")

	gitHttpUploadCounter.WithLabelValues(repo, file).Inc()

	if service != "" && (service == git.UploadPackService || service == git.ReceivePackService) {
		// Smart HTTP
		var refs bytes.Buffer
		cmd := git.ServiceCommand{
			Stdout: &refs,
			Dir:    dir,
			Args:   []string{"--stateless-rpc", "--advertise-refs"},
		}

		if len(version) != 0 {
			cmd.Env = append(cmd.Env, fmt.Sprintf("GIT_PROTOCOL=%s", version))
		}

		if err := service.Handler(ctx, cmd); err != nil {
			logger.Errorf("error executing service: %s", err)
			renderNotFound(w)
			return
		}

		hdrNocache(w)
		w.Header().Set("Content-Type", fmt.Sprintf("application/x-%s-advertisement", service))
		w.WriteHeader(http.StatusOK)
		if len(version) == 0 {
			git.WritePktline(w, "# service="+service.String())
		}

		w.Write(refs.Bytes()) // nolint: errcheck
	} else {
		// Dumb HTTP
		updateServerInfo(ctx, dir) // nolint: errcheck
		hdrNocache(w)
		sendFile("text/plain; charset=utf-8", w, r)
	}
}

func getInfoPacks(w http.ResponseWriter, r *http.Request) {
	hdrCacheForever(w)
	sendFile("text/plain; charset=utf-8", w, r)
}

func getLooseObject(w http.ResponseWriter, r *http.Request) {
	hdrCacheForever(w)
	sendFile("application/x-git-loose-object", w, r)
}

func getPackFile(w http.ResponseWriter, r *http.Request) {
	hdrCacheForever(w)
	sendFile("application/x-git-packed-objects", w, r)
}

func getIdxFile(w http.ResponseWriter, r *http.Request) {
	hdrCacheForever(w)
	sendFile("application/x-git-packed-objects-toc", w, r)
}

func getTextFile(w http.ResponseWriter, r *http.Request) {
	hdrNocache(w)
	sendFile("text/plain", w, r)
}

func sendFile(contentType string, w http.ResponseWriter, r *http.Request) {
	dir, file := pat.Param(r, "dir"), pat.Param(r, "file")
	reqFile := filepath.Join(dir, file)

	f, err := os.Stat(reqFile)
	if os.IsNotExist(err) {
		renderNotFound(w)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", f.Size()))
	w.Header().Set("Last-Modified", f.ModTime().Format(http.TimeFormat))
	http.ServeFile(w, r, reqFile)
}

func getServiceType(r *http.Request) git.Service {
	service := r.FormValue("service")
	if !strings.HasPrefix(service, "git-") {
		return ""
	}

	return git.Service(service)
}

func isSmart(r *http.Request, service git.Service) bool {
	if r.Header.Get("Content-Type") == fmt.Sprintf("application/x-%s-request", service) {
		return true
	}
	return false
}

func updateServerInfo(ctx context.Context, dir string) error {
	return gitb.UpdateServerInfo(ctx, dir)
}

// HTTP error response handling functions

func renderMethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	if r.Proto == "HTTP/1.1" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method Not Allowed")) // nolint: errcheck
	} else {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad Request")) // nolint: errcheck
	}
}

func renderNotFound(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("Not Found")) // nolint: errcheck
}

func renderUnauthorized(w http.ResponseWriter) {
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte("Unauthorized")) // nolint: errcheck
}

func renderForbidden(w http.ResponseWriter) {
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte("Forbidden")) // nolint: errcheck
}

func renderInternalServerError(w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("Internal Server Error")) // nolint: errcheck
}

// Header writing functions

func hdrNocache(w http.ResponseWriter) {
	w.Header().Set("Expires", "Fri, 01 Jan 1980 00:00:00 GMT")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Cache-Control", "no-cache, max-age=0, must-revalidate")
}

func hdrCacheForever(w http.ResponseWriter) {
	now := time.Now().Unix()
	expires := now + 31536000
	w.Header().Set("Date", fmt.Sprintf("%d", now))
	w.Header().Set("Expires", fmt.Sprintf("%d", expires))
	w.Header().Set("Cache-Control", "public, max-age=31536000")
}
