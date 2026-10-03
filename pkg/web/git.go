package web

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"charm.land/log/v2"
	gitb "github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/git"
	"github.com/charmbracelet/soft-serve/pkg/lfs"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/utils"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// GitRoute is a route for git services.
type GitRoute struct {
	method  []string
	handler http.HandlerFunc
	path    string
}

var _ http.Handler = GitRoute{}

// ServeHTTP implements http.Handler.
func (g GitRoute) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var hasMethod bool
	for _, m := range g.method {
		if m == r.Method {
			hasMethod = true
			break
		}
	}

	if !hasMethod {
		renderMethodNotAllowed(w, r)
		return
	}

	g.handler(w, r)
}

var (
	//nolint:revive
	gitHttpReceiveCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "soft_serve",
		Subsystem: "http",
		Name:      "git_receive_pack_total",
		Help:      "The total number of git push requests",
	}, []string{"repo"})

	//nolint:revive
	gitHttpUploadCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "soft_serve",
		Subsystem: "http",
		Name:      "git_upload_pack_total",
		Help:      "The total number of git fetch/pull requests",
	}, []string{"repo", "file"})
)

func withParams(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		cfg := config.FromContext(ctx)
		vars := mux.Vars(r)
		repo := vars["repo"]

		// Construct "file" param from path
		vars["file"] = strings.TrimPrefix(r.URL.Path, "/"+repo+"/")

		// Set service type
		switch {
		case strings.HasSuffix(r.URL.Path, git.UploadPackService.String()):
			vars["service"] = git.UploadPackService.String()
		case strings.HasSuffix(r.URL.Path, git.ReceivePackService.String()):
			vars["service"] = git.ReceivePackService.String()
		}

		repo = utils.SanitizeRepo(repo)
		vars["repo"] = repo
		vars["dir"] = filepath.Join(cfg.DataPath, "repos", repo+".git")

		// Add repo suffix (.git)
		r.URL.Path = fmt.Sprintf("%s.git/%s", repo, vars["file"])
		r = mux.SetURLVars(r, vars)

		next.ServeHTTP(w, r)
	})
}

// GitController is a router for git services.
func GitController(_ context.Context, r *mux.Router) {
	basePrefix := "/{repo:.*}"
	for _, route := range gitRoutes {
		// NOTE: withParam must always be the outermost wrapper, otherwise the
		// request vars will not be set.
		r.Handle(basePrefix+route.path, withParams(withAccess(route)))
	}

	// Handle go-get
	r.Handle(basePrefix, withParams(withCaller(http.HandlerFunc(GoGetHandler)))).Methods(http.MethodGet)
}

var gitRoutes = []GitRoute{
	// Git services
	// These routes don't handle authentication/authorization.
	// This is handled through wrapping the handlers for each route.
	// See below (withAccess).
	{
		method:  []string{http.MethodPost},
		handler: serviceRpc,
		path:    "/{service:(?:git-upload-archive|git-upload-pack|git-receive-pack)$}",
	},
	{
		method:  []string{http.MethodGet},
		handler: getInfoRefs,
		path:    "/info/refs",
	},
	{
		method:  []string{http.MethodGet},
		handler: getTextFile,
		path:    "/{_:(?:HEAD|objects/info/alternates|objects/info/http-alternates|objects/info/[^/]*)$}",
	},
	{
		method:  []string{http.MethodGet},
		handler: getInfoPacks,
		path:    "/objects/info/packs",
	},
	{
		method:  []string{http.MethodGet},
		handler: getLooseObject,
		path:    "/objects/{_:[0-9a-f]{2}/[0-9a-f]{38}$}",
	},
	{
		method:  []string{http.MethodGet},
		handler: getPackFile,
		path:    "/objects/pack/{_:pack-[0-9a-f]{40}\\.pack$}",
	},
	{
		method:  []string{http.MethodGet},
		handler: getIdxFile,
		path:    "/objects/pack/{_:pack-[0-9a-f]{40}\\.idx$}",
	},
	// Git LFS
	{
		method:  []string{http.MethodPost},
		handler: serviceLfsBatch,
		path:    "/info/lfs/objects/batch",
	},
	{
		// Git LFS basic object handler
		method:  []string{http.MethodGet, http.MethodPut},
		handler: serviceLfsBasic,
		path:    "/info/lfs/objects/basic/{oid:[0-9a-f]{64}$}",
	},
	{
		method:  []string{http.MethodPost},
		handler: serviceLfsBasicVerify,
		path:    "/info/lfs/objects/basic/verify",
	},
	// Git LFS locks
	{
		method:  []string{http.MethodPost, http.MethodGet},
		handler: serviceLfsLocks,
		path:    "/info/lfs/locks",
	},
	{
		method:  []string{http.MethodPost},
		handler: serviceLfsLocksVerify,
		path:    "/info/lfs/locks/verify",
	},
	{
		method:  []string{http.MethodPost},
		handler: serviceLfsLocksDelete,
		path:    "/info/lfs/locks/{lock_id:[0-9]+}/unlock",
	},
}

func askCredentials(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("WWW-Authenticate", `Basic realm="Git" charset="UTF-8", Token, Bearer`)
	w.Header().Set("LFS-Authenticate", `Basic realm="Git LFS" charset="UTF-8", Token, Bearer`)
}

// identify stores the requested repository and the authenticated caller in
// the request context, and returns the authentication error, if any.
func identify(r *http.Request) (*http.Request, proto.Repository, error) {
	ctx := r.Context()
	logger := log.FromContext(ctx)
	be := backend.FromContext(ctx)

	// A missing repository is not an error here; pushes create it.
	repo, _ := be.Repository(ctx, mux.Vars(r)["repo"])
	ctx = proto.WithRepositoryContext(ctx, repo)
	r = r.WithContext(ctx)

	user, err := authenticate(r)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidToken):
		case errors.Is(err, proto.ErrUserNotFound):
		default:
			logger.Error("failed to authenticate", "err", err)
		}
	}

	if user != nil {
		logger.Debug("authenticated", "username", user.Username())
	}

	return r.WithContext(proto.WithUserContext(ctx, user)), repo, err
}

// badCredentials reports whether the caller sent credentials that failed.
func badCredentials(err error) bool {
	return errors.Is(err, ErrInvalidToken) || errors.Is(err, ErrInvalidPassword)
}

// denyRead answers a caller who may not read the repository. Missing and
// unreadable repositories get the same answer so neither is revealed.
func denyRead(w http.ResponseWriter, r *http.Request, err error, lfsJSON bool) {
	switch {
	case badCredentials(err) && lfsJSON:
		renderJSON(w, http.StatusForbidden, lfs.ErrorResponse{Message: "bad credentials"})
	case badCredentials(err):
		renderForbidden(w, r)
	case proto.UserFromContext(r.Context()) == nil && lfsJSON:
		askCredentials(w, r)
		renderJSON(w, http.StatusUnauthorized, lfs.ErrorResponse{Message: "credentials needed"})
	case proto.UserFromContext(r.Context()) == nil:
		askCredentials(w, r)
		renderUnauthorized(w, r)
	case lfsJSON:
		renderJSON(w, http.StatusNotFound, lfs.ErrorResponse{Message: "repository not found"})
	default:
		renderNotFound(w, r)
	}
}

// lfsNeedsWrite reports whether an LFS request needs write access.
// https://github.com/git-lfs/git-lfs/blob/main/docs/api/locking.md
func lfsNeedsWrite(file, method string) bool {
	switch {
	case strings.HasPrefix(file, "info/lfs/locks"):
		return strings.HasSuffix(file, "lfs/locks") ||
			strings.HasSuffix(file, "lfs/locks/verify") ||
			strings.HasSuffix(file, "/unlock") && method == http.MethodPost
	case strings.HasPrefix(file, "info/lfs/objects/basic"):
		return method == http.MethodPut
	}
	return false
}

// requireCaller rejects anonymous requests when keyless access is disabled.
func requireCaller(w http.ResponseWriter, r *http.Request) bool {
	ctx := r.Context()
	if proto.UserFromContext(ctx) == nil && !backend.FromContext(ctx).AllowKeyless(ctx) {
		askCredentials(w, r)
		renderUnauthorized(w, r)
		return false
	}
	return true
}

// withCaller authenticates the caller without authorizing a repository.
// Handlers behind it must check access to any repository they reveal.
func withCaller(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r, _, err := identify(r)
		if !requireCaller(w, r) {
			return
		}

		if badCredentials(err) {
			renderForbidden(w, r)
			return
		}

		next.ServeHTTP(w, r)
	}
}

// withAccess authenticates the caller and authorizes them for the requested
// repository.
func withAccess(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r, repo, err := identify(r)
		if !requireCaller(w, r) {
			return
		}

		ctx := r.Context()
		cfg := config.FromContext(ctx)
		logger := log.FromContext(ctx)
		be := backend.FromContext(ctx)
		repoName := mux.Vars(r)["repo"]
		user := proto.UserFromContext(ctx)

		service := git.Service(mux.Vars(r)["service"])
		if service == "" {
			// Get service from request params
			service = getServiceType(r)
		}

		accessLevel := be.AccessLevelForUser(ctx, repoName, user)
		ctx = access.WithContext(ctx, accessLevel)
		r = r.WithContext(ctx)

		file := mux.Vars(r)["file"]

		// The LFS case must stay ahead of the service cases. "file" is derived
		// from the request path, but "service" can come from a query parameter
		// the caller controls, so an LFS request is authorized as LFS no
		// matter what service it claims to be.
		switch {
		case strings.HasPrefix(file, "info/lfs"):
			if !cfg.LFS.Enabled {
				logger.Debug("LFS is not enabled, skipping")
				renderNotFound(w, r)
				return
			}

			if repo == nil || accessLevel < access.ReadOnlyAccess {
				denyRead(w, r, err, true)
				return
			}

			if lfsNeedsWrite(file, r.Method) && accessLevel < access.ReadWriteAccess {
				renderJSON(w, http.StatusForbidden, lfs.ErrorResponse{
					Message: "write access required",
				})
				return
			}

		case service == git.ReceivePackService:
			if accessLevel < access.ReadWriteAccess {
				askCredentials(w, r)
				renderUnauthorized(w, r)
				return
			}

			// Create the repo if it doesn't exist.
			if repo == nil {
				repo, err = be.CreateRepository(ctx, repoName, user, proto.RepositoryOptions{})
				if err != nil {
					logger.Error("failed to create repository", "repo", repoName, "err", err)
					renderInternalServerError(w, r)
					return
				}

				ctx = proto.WithRepositoryContext(ctx, repo)
				r = r.WithContext(ctx)
			}
		}

		if badCredentials(err) {
			renderForbidden(w, r)
			return
		}

		if repo == nil || accessLevel < access.ReadOnlyAccess {
			denyRead(w, r, err, false)
			return
		}

		next.ServeHTTP(w, r)
	}
}

//nolint:revive
func serviceRpc(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cfg := config.FromContext(ctx)
	logger := log.FromContext(ctx)
	service, dir, repoName := git.Service(mux.Vars(r)["service"]), mux.Vars(r)["dir"], mux.Vars(r)["repo"]

	if !isSmart(r, service) {
		renderForbidden(w, r)
		return
	}

	if service == git.ReceivePackService {
		gitHttpReceiveCounter.WithLabelValues(repoName)
	}

	w.Header().Set("Content-Type", fmt.Sprintf("application/x-%s-result", service))
	w.Header().Set("Connection", "Keep-Alive")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)

	version := r.Header.Get("Git-Protocol")

	cmd := git.ServiceCommand{
		Dir: dir,
	}

	switch service {
	case git.UploadPackService, git.ReceivePackService:
		cmd.Args = append(cmd.Args, "--stateless-rpc")
	}

	user := proto.UserFromContext(ctx)
	cmd.Env = cfg.Environ()
	cmd.Env = append(cmd.Env, []string{
		"SOFT_SERVE_REPO_NAME=" + repoName,
		"SOFT_SERVE_REPO_PATH=" + dir,
		"SOFT_SERVE_LOG_PATH=" + filepath.Join(cfg.DataPath, "log", "hooks.log"),
	}...)
	if user != nil {
		cmd.Env = append(cmd.Env, []string{
			"SOFT_SERVE_USERNAME=" + user.Username(),
		}...)
	}
	if len(version) != 0 {
		cmd.Env = append(cmd.Env, []string{
			fmt.Sprintf("GIT_PROTOCOL=%s", version),
		}...)
	}

	var (
		err    error
		reader io.ReadCloser
	)

	// Handle gzip encoding
	reader = r.Body
	switch r.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(reader)
		if err != nil {
			logger.Errorf("failed to create gzip reader: %v", err)
			renderInternalServerError(w, r)
			return
		}
		defer reader.Close() //nolint: errcheck
	}

	cmd.Stdin = reader
	cmd.Stdout = w

	if err := service.Handler(ctx, cmd); err != nil {
		logger.Errorf("failed to handle service: %v", err)
		return
	}

	if service == git.ReceivePackService {
		if err := git.EnsureDefaultBranch(ctx, cmd.Dir); err != nil {
			logger.Errorf("failed to ensure default branch: %s", err)
		}
	}
}

func getInfoRefs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cfg := config.FromContext(ctx)
	dir, repoName, file := mux.Vars(r)["dir"], mux.Vars(r)["repo"], mux.Vars(r)["file"]
	service := getServiceType(r)
	protocol := r.Header.Get("Git-Protocol")

	gitHttpUploadCounter.WithLabelValues(repoName, file).Inc()

	if service != "" && (service == git.UploadPackService || service == git.ReceivePackService) {
		// Smart HTTP
		var refs bytes.Buffer
		cmd := git.ServiceCommand{
			Stdout: &refs,
			Dir:    dir,
			Args:   []string{"--stateless-rpc", "--advertise-refs"},
		}

		user := proto.UserFromContext(ctx)
		cmd.Env = cfg.Environ()
		cmd.Env = append(cmd.Env, []string{
			"SOFT_SERVE_REPO_NAME=" + repoName,
			"SOFT_SERVE_REPO_PATH=" + dir,
			"SOFT_SERVE_LOG_PATH=" + filepath.Join(cfg.DataPath, "log", "hooks.log"),
		}...)
		if user != nil {
			cmd.Env = append(cmd.Env, []string{
				"SOFT_SERVE_USERNAME=" + user.Username(),
			}...)
		}
		if len(protocol) != 0 {
			cmd.Env = append(cmd.Env, fmt.Sprintf("GIT_PROTOCOL=%s", protocol))
		}

		var version int
		for _, p := range strings.Split(protocol, ":") {
			if strings.HasPrefix(p, "version=") {
				if v, _ := strconv.Atoi(p[8:]); v > version {
					version = v
				}
			}
		}

		if err := service.Handler(ctx, cmd); err != nil {
			renderNotFound(w, r)
			return
		}

		hdrNocache(w)
		w.Header().Set("Content-Type", fmt.Sprintf("application/x-%s-advertisement", service))
		w.WriteHeader(http.StatusOK)
		if version < 2 {
			git.WritePktline(w, "# service="+service.String()) //nolint: errcheck
		}
		w.Write(refs.Bytes()) //nolint: errcheck
	} else {
		// Dumb HTTP
		updateServerInfo(ctx, dir) //nolint: errcheck
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
	dir, file := mux.Vars(r)["dir"], mux.Vars(r)["file"]
	reqFile := filepath.Join(dir, file)

	f, err := os.Stat(reqFile)
	if os.IsNotExist(err) {
		renderNotFound(w, r)
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
	contentType := r.Header.Get("Content-Type")
	return strings.HasPrefix(contentType, fmt.Sprintf("application/x-%s-request", service))
}

func updateServerInfo(ctx context.Context, dir string) error {
	return gitb.UpdateServerInfo(ctx, dir)
}

// HTTP error response handling functions

func renderBadRequest(w http.ResponseWriter, r *http.Request) {
	renderStatus(http.StatusBadRequest)(w, r)
}

func renderMethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	if r.Proto == "HTTP/1.1" {
		renderStatus(http.StatusMethodNotAllowed)(w, r)
	} else {
		renderBadRequest(w, r)
	}
}

func renderNotFound(w http.ResponseWriter, r *http.Request) {
	renderStatus(http.StatusNotFound)(w, r)
}

func renderUnauthorized(w http.ResponseWriter, r *http.Request) {
	renderStatus(http.StatusUnauthorized)(w, r)
}

func renderForbidden(w http.ResponseWriter, r *http.Request) {
	renderStatus(http.StatusForbidden)(w, r)
}

func renderInternalServerError(w http.ResponseWriter, r *http.Request) {
	renderStatus(http.StatusInternalServerError)(w, r)
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
