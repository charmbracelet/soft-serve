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

// gitSuffixMiddleware rewrites request paths so that repos are accessible
// without the .git suffix when cfg.HTTP.StripGitSuffix is true.
// It inserts ".git" before any recognised git sub-path so that all
// downstream handlers continue to see the canonical /<name>.git/... form.
// gitSuffixMiddleware handles the StripGitSuffix config option. When enabled,
// it INSERTS ".git" into URL paths for clients that omit the suffix — the
// flag name describes the client-side behaviour (clients may strip ".git")
// while this middleware normalises paths to the canonical ".git"-suffixed form.
func gitSuffixMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	// Known sub-paths that immediately follow the repo segment.
	gitSubPaths := []string{
		"/info/refs",
		"/git-upload-pack",
		"/git-receive-pack",
		"/git-upload-archive",
		"/HEAD",
		"/objects/",
		"/info/lfs/",
		"/raw/",
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.HTTP.StripGitSuffix && !strings.Contains(r.URL.Path, ".git/") && !strings.HasSuffix(r.URL.Path, ".git") {
				p := r.URL.Path
				for _, sub := range gitSubPaths {
					if idx := strings.Index(p, sub); idx > 0 {
						// Insert .git before the sub-path.
						newPath := p[:idx] + ".git" + p[idx:]
						r2 := r.Clone(r.Context())
						r2.URL.Path = newPath
						if r.URL.RawPath != "" {
							rawIdx := strings.Index(r.URL.RawPath, sub)
							if rawIdx < 0 {
								rawIdx = idx // fallback: decoded and encoded same length
							}
							r2.URL.RawPath = r.URL.RawPath[:rawIdx] + ".git" + r.URL.RawPath[rawIdx:]
						}
						next.ServeHTTP(w, r2)
						return
					}
				}
				// No sub-path found — path is bare /<repo> (e.g. go-get).
				// Append .git so withParams can strip it back to the repo name.
				if p != "/" {
					r2 := r.Clone(r.Context())
					r2.URL.Path = strings.TrimSuffix(p, "/") + ".git"
					next.ServeHTTP(w, r2)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func withParams(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		cfg := config.FromContext(ctx)
		vars := mux.Vars(r)
		repo := vars["repo"]

		// repo still has the .git suffix here; vars["file"] is the path component after /<repo>/.
		// SanitizeRepo (applied below) strips the .git suffix from vars["repo"] only.
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
	r.Handle(basePrefix, withParams(withAccess(http.HandlerFunc(GoGetHandler)))).Methods(http.MethodGet)
}

var gitRoutes = []GitRoute{
	// Raw file content endpoint.
	// Must be registered before the git-protocol routes so that
	// /raw/... paths are not swallowed by the git object handlers.
	{
		method:  []string{http.MethodGet},
		handler: getRawBlob,
		path:    "/raw/{ref}/{filepath:.*}",
	},
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

// withAccess handles auth.
func withAccess(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		cfg := config.FromContext(ctx)
		logger := log.FromContext(ctx)
		be := backend.FromContext(ctx)

		// Store repository in context
		// We're not checking for errors here because we want to allow
		// repo creation on the fly.
		repoName := mux.Vars(r)["repo"]
		repo, _ := be.Repository(ctx, repoName)
		ctx = proto.WithRepositoryContext(ctx, repo)
		r = r.WithContext(ctx)

		user, err := authenticate(r)
		if err != nil {
			switch {
			case errors.Is(err, errInvalidToken):
			case errors.Is(err, errInvalidPassword):
			case errors.Is(err, proto.ErrUserNotFound):
			default:
				logger.Error("failed to authenticate", "err", err)
			}
		}

		if user == nil && !be.AllowKeyless(ctx) {
			askCredentials(w, r)
			renderUnauthorized(w, r)
			return
		}

		// Store user in context
		ctx = proto.WithUserContext(ctx, user)
		r = r.WithContext(ctx)

		if user != nil {
			logger.Debug("authenticated", "username", user.Username())
		}

		service := git.Service(mux.Vars(r)["service"])
		if service == "" {
			// Get service from request params
			service = getServiceType(r)
		}

		accessLevel := be.AccessLevelForUser(ctx, repoName, user)
		ctx = access.WithContext(ctx, accessLevel)
		r = r.WithContext(ctx)

		file := mux.Vars(r)["file"]

		// We only allow these services to proceed any other services should return 403
		// - git-upload-pack
		// - git-receive-pack
		// - git-lfs
		//
		// Routes that carry no service var and no info/lfs prefix (e.g. the raw
		// blob endpoint, go-get) have no matching case in this switch (no default
		// case). They fall through to the catch-all access check below which
		// enforces ReadOnlyAccess via renderNotFound.
		switch {
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

			fallthrough
		case service == git.UploadPackService || service == git.UploadArchiveService:
			// Return 404 for missing repos regardless of any credential error to
			// prevent repository enumeration: an unauthenticated client must not
			// be able to distinguish "repo doesn't exist" from "access denied".
			if repo == nil {
				renderNotFound(w, r)
				return
			}
			// Repo exists — now check credentials and access level.
			if errors.Is(err, errInvalidToken) || errors.Is(err, errInvalidPassword) {
				renderForbidden(w, r)
				return
			}
			if accessLevel < access.ReadOnlyAccess {
				askCredentials(w, r)
				renderUnauthorized(w, r)
				return
			}

		case strings.HasPrefix(file, "info/lfs"):
			if !cfg.LFS.Enabled {
				logger.Debug("LFS is not enabled, skipping")
				renderNotFound(w, r)
				return
			}

			switch {
			case strings.HasPrefix(file, "info/lfs/locks"):
				switch {
				case strings.HasSuffix(file, "lfs/locks") && r.Method == http.MethodPost,
					strings.HasSuffix(file, "/unlock") && r.Method == http.MethodPost,
					strings.HasSuffix(file, "lfs/locks/verify"):
					// Create lock, delete lock, and verify require write access.
					// https://github.com/git-lfs/git-lfs/blob/main/docs/api/locking.md#unauthorized-response-2
					if accessLevel < access.ReadWriteAccess {
						renderJSON(w, r, http.StatusForbidden, lfs.ErrorResponse{
							Message: "write access required",
						})
						return
					}
				case strings.HasSuffix(file, "lfs/locks") && r.Method == http.MethodGet:
					// List locks only requires read access.
					if accessLevel < access.ReadOnlyAccess {
						renderJSON(w, r, http.StatusForbidden, lfs.ErrorResponse{
							Message: "read access required",
						})
						return
					}
				}
			case strings.HasPrefix(file, "info/lfs/objects/basic"):
				switch r.Method {
				case http.MethodPut:
					// Basic upload
					if accessLevel < access.ReadWriteAccess {
						renderJSON(w, r, http.StatusForbidden, lfs.ErrorResponse{
							Message: "write access required",
						})
						return
					}
				case http.MethodGet:
					// Basic download
				case http.MethodPost:
					// No additional access check here — basic verify is a read-only metadata operation;
					// auth is already enforced by the outer withAccess middleware.
				}
			}

			if accessLevel < access.ReadOnlyAccess {
				if repo == nil {
					renderJSON(w, r, http.StatusNotFound, lfs.ErrorResponse{
						Message: "repository not found",
					})
				} else if errors.Is(err, errInvalidToken) || errors.Is(err, errInvalidPassword) {
					renderJSON(w, r, http.StatusForbidden, lfs.ErrorResponse{
						Message: "bad credentials",
					})
				} else {
					askCredentials(w, r)
					renderJSON(w, r, http.StatusUnauthorized, lfs.ErrorResponse{
						Message: "credentials needed",
					})
				}
				return
			}
		}

		switch {
		case r.URL.Query().Get("go-get") == "1" && repo != nil && (accessLevel >= access.ReadOnlyAccess || cfg.AllowPublicGoGet):
			// Allow go-get requests to pass through.
			// Note: when AllowPublicGoGet is true, unauthenticated clients can
			// learn that a private repo exists by comparing go-get (200) vs a
			// regular (404) response. This is an accepted trade-off for go module
			// discoverability.
			break
		case errors.Is(err, errInvalidToken), errors.Is(err, errInvalidPassword):
			// return 403 when bad credentials are provided
			renderForbidden(w, r)
			return
		case repo == nil, accessLevel < access.ReadOnlyAccess:
			// Don't hint that the repo exists if the user doesn't have access
			renderNotFound(w, r)
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
		gitHttpReceiveCounter.WithLabelValues(repoName).Inc()
	}

	gitProtocol := r.Header.Get("Git-Protocol")
	// Sanitize: reject any control characters to prevent env var injection.
	for _, c := range gitProtocol {
		if c < 0x20 || c == 0x7f {
			gitProtocol = ""
			break
		}
	}

	var stdout bytes.Buffer
	cmd := git.ServiceCommand{
		Stdout: &stdout,
		Dir:    dir,
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
		// user.Username() is validated by ValidateUsername (letters/digits/hyphens only) — no injection risk.
		cmd.Env = append(cmd.Env, []string{
			"SOFT_SERVE_USERNAME=" + user.Username(),
		}...)
	}
	if gitProtocol != "" {
		cmd.Env = append(cmd.Env, "GIT_PROTOCOL="+gitProtocol)
	}

	// maxReceivePackSize is the maximum size of a git-receive-pack request body (10 GiB).
	const maxReceivePackSize = 10 * 1024 * 1024 * 1024

	// maxUploadPackSize limits upload-pack request bodies. upload-pack only receives
	// "want" and "have" lines (no actual object data), so 256 MiB is generous.
	const maxUploadPackSize = 256 * 1024 * 1024

	var reader io.ReadCloser

	// Cap the body for both services to prevent unbounded reads.
	if service == git.ReceivePackService {
		r.Body = http.MaxBytesReader(w, r.Body, maxReceivePackSize)
	} else {
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadPackSize)
	}

	// Handle gzip encoding before committing to 200 OK so we can still
	// return an error status if the gzip stream is malformed.
	reader = r.Body
	switch r.Header.Get("Content-Encoding") {
	case "gzip":
		gzReader, gzErr := gzip.NewReader(reader)
		if gzErr != nil {
			logger.Errorf("failed to create gzip reader: %v", gzErr)
			renderInternalServerError(w, r)
			return
		}
		defer gzReader.Close() //nolint: errcheck
		if service == git.ReceivePackService {
			reader = io.NopCloser(io.LimitReader(gzReader, maxReceivePackSize))
		} else {
			reader = io.NopCloser(io.LimitReader(gzReader, maxUploadPackSize))
		}
	}

	// WriteHeader is deferred until after all request parsing so that
	// parsing errors (e.g. bad gzip stream above) can return a non-200 status.
	w.Header().Set("Content-Type", fmt.Sprintf("application/x-%s-result", service))
	w.Header().Set("Connection", "Keep-Alive")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)

	cmd.Stdin = reader
	cmd.Stdout = &flushResponseWriter{w}

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

// Handle buffered output
// Useful when using proxies
type flushResponseWriter struct {
	http.ResponseWriter
}

const flushBufSize = 32 * 1024

func (f *flushResponseWriter) ReadFrom(r io.Reader) (int64, error) {
	flusher := http.NewResponseController(f.ResponseWriter)

	var n int64
	p := make([]byte, flushBufSize)
	for {
		// Read first, then check error — a Read may return n > 0 bytes AND
		// io.EOF simultaneously (per the io.Reader contract), so we must
		// write any bytes before acting on the error.
		nRead, readErr := r.Read(p)
		if nRead > 0 {
			nWrite, err := f.ResponseWriter.Write(p[:nRead])
			n += int64(nWrite)
			if err != nil {
				return n, err
			}
			if nWrite < nRead {
				return n, io.ErrShortWrite
			}
			// ResponseWriter must support http.Flusher to handle buffered output.
			if err := flusher.Flush(); err != nil {
				return n, fmt.Errorf("error while flush: %w", err)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return n, readErr
		}
	}

	return n, nil
}

func getInfoRefs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cfg := config.FromContext(ctx)
	dir, repoName, file := mux.Vars(r)["dir"], mux.Vars(r)["repo"], mux.Vars(r)["file"]
	service := getServiceType(r)
	protocol := r.Header.Get("Git-Protocol")
	// Sanitize: reject any control characters to prevent env var injection.
	for _, c := range protocol {
		if c < 0x20 || c == 0x7f {
			protocol = ""
			break
		}
	}

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
			// user.Username() is validated by ValidateUsername (letters/digits/hyphens only) — no injection risk.
			cmd.Env = append(cmd.Env, []string{
				"SOFT_SERVE_USERNAME=" + user.Username(),
			}...)
		}
		if protocol != "" {
			cmd.Env = append(cmd.Env, "GIT_PROTOCOL="+protocol)
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
		// Use flushResponseWriter so that the pktline header and ref
		// advertisement are flushed through buffering proxies promptly.
		fw := &flushResponseWriter{w}
		if version < 2 {
			git.WritePktline(fw, "# service="+service.String()) //nolint: errcheck
		}
		fw.Write(refs.Bytes()) //nolint: errcheck
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

	// Guard against path traversal.
	root := dir + string(filepath.Separator)
	if !strings.HasPrefix(reqFile, root) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if rel, err := filepath.Rel(dir, reqFile); err != nil || strings.HasPrefix(rel, "..") {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	// Use Lstat to detect symlinks before opening. A narrow TOCTOU window
	// remains between Lstat and Open; it is acceptable because git repository
	// internals are server-controlled and not writable by pushing users.
	fi, err := os.Lstat(reqFile)
	if os.IsNotExist(err) {
		renderNotFound(w, r)
		return
	}
	if err != nil {
		renderInternalServerError(w, r)
		return
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		renderNotFound(w, r)
		return
	}

	// Open the file ourselves and serve via http.ServeContent to avoid the
	// second internal os.Open that http.ServeFile performs (which follows
	// symlinks). This narrows the TOCTOU window to the Lstat→Open interval.
	fd, fdErr := os.Open(reqFile)
	if fdErr != nil {
		if os.IsNotExist(fdErr) {
			renderNotFound(w, r)
		} else {
			renderInternalServerError(w, r)
		}
		return
	}
	defer fd.Close() //nolint:errcheck

	w.Header().Set("Content-Type", contentType)
	http.ServeContent(w, r, reqFile, fi.ModTime(), fd)
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

const oneYearSeconds = 31536000

func hdrCacheForever(w http.ResponseWriter) {
	now := time.Now().Unix()
	expires := now + oneYearSeconds
	w.Header().Set("Date", time.Unix(now, 0).UTC().Format(http.TimeFormat))
	w.Header().Set("Expires", time.Unix(expires, 0).UTC().Format(http.TimeFormat))
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", oneYearSeconds))
}
