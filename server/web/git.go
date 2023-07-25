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
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	gitb "github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/server/access"
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/git"
	"github.com/charmbracelet/soft-serve/server/lfs"
	"github.com/charmbracelet/soft-serve/server/proto"
	"github.com/charmbracelet/soft-serve/server/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"goji.io/pat"
	"goji.io/pattern"
)

// GitRoute is a route for git services.
type GitRoute struct {
	method  []string
	pattern *regexp.Regexp
	handler http.HandlerFunc
}

var _ Route = GitRoute{}

// Match implements goji.Pattern.
func (g GitRoute) Match(r *http.Request) *http.Request {
	re := g.pattern
	ctx := r.Context()
	cfg := config.FromContext(ctx)
	if m := re.FindStringSubmatch(r.URL.Path); m != nil {
		// This finds the Git objects & packs filenames in the URL.
		file := strings.Replace(r.URL.Path, m[1]+"/", "", 1)
		repo := utils.SanitizeRepo(m[1])

		var service git.Service
		var oid string    // LFS object ID
		var lockID string // LFS lock ID
		switch {
		case strings.HasSuffix(r.URL.Path, git.UploadPackService.String()):
			service = git.UploadPackService
		case strings.HasSuffix(r.URL.Path, git.ReceivePackService.String()):
			service = git.ReceivePackService
		case len(m) > 2:
			if strings.HasPrefix(file, "info/lfs/objects/basic/") {
				oid = m[2]
			} else if strings.HasPrefix(file, "info/lfs/locks/") && strings.HasSuffix(file, "/unlock") {
				lockID = m[2]
			}
			fallthrough
		case strings.HasPrefix(file, "info/lfs"):
			service = gitLfsService
		}

		ctx = context.WithValue(ctx, pattern.Variable("lock_id"), lockID)
		ctx = context.WithValue(ctx, pattern.Variable("oid"), oid)
		ctx = context.WithValue(ctx, pattern.Variable("service"), service.String())
		ctx = context.WithValue(ctx, pattern.Variable("dir"), filepath.Join(cfg.DataPath, "repos", repo+".git"))
		ctx = context.WithValue(ctx, pattern.Variable("repo"), repo)
		ctx = context.WithValue(ctx, pattern.Variable("file"), file)

		return r.WithContext(ctx)
	}

	return nil
}

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

var (
	serviceRpcMatcher            = regexp.MustCompile("(.*?)/(?:git-upload-pack|git-receive-pack)$") // nolint: revive
	getInfoRefsMatcher           = regexp.MustCompile("(.*?)/info/refs$")
	getTextFileMatcher           = regexp.MustCompile("(.*?)/(?:HEAD|objects/info/alternates|objects/info/http-alternates|objects/info/[^/]*)$")
	getInfoPacksMatcher          = regexp.MustCompile("(.*?)/objects/info/packs$")
	getLooseObjectMatcher        = regexp.MustCompile("(.*?)/objects/[0-9a-f]{2}/[0-9a-f]{38}$")
	getPackFileMatcher           = regexp.MustCompile(`(.*?)/objects/pack/pack-[0-9a-f]{40}\.pack$`)
	getIdxFileMatcher            = regexp.MustCompile(`(.*?)/objects/pack/pack-[0-9a-f]{40}\.idx$`)
	serviceLfsBatchMatcher       = regexp.MustCompile("(.*?)/info/lfs/objects/batch$")
	serviceLfsBasicMatcher       = regexp.MustCompile("(.*?)/info/lfs/objects/basic/([0-9a-f]{64})$")
	serviceLfsBasicVerifyMatcher = regexp.MustCompile("(.*?)/info/lfs/objects/basic/verify$")
)

var gitRoutes = []GitRoute{
	// Git services
	// These routes don't handle authentication/authorization.
	// This is handled through wrapping the handlers for each route.
	// See below (withAccess).
	{
		pattern: serviceRpcMatcher,
		method:  []string{http.MethodPost},
		handler: serviceRpc,
	},
	{
		pattern: getInfoRefsMatcher,
		method:  []string{http.MethodGet},
		handler: getInfoRefs,
	},
	{
		pattern: getTextFileMatcher,
		method:  []string{http.MethodGet},
		handler: getTextFile,
	},
	{
		pattern: getTextFileMatcher,
		method:  []string{http.MethodGet},
		handler: getTextFile,
	},
	{
		pattern: getInfoPacksMatcher,
		method:  []string{http.MethodGet},
		handler: getInfoPacks,
	},
	{
		pattern: getLooseObjectMatcher,
		method:  []string{http.MethodGet},
		handler: getLooseObject,
	},
	{
		pattern: getPackFileMatcher,
		method:  []string{http.MethodGet},
		handler: getPackFile,
	},
	{
		pattern: getIdxFileMatcher,
		method:  []string{http.MethodGet},
		handler: getIdxFile,
	},
	// Git LFS
	{
		pattern: serviceLfsBatchMatcher,
		method:  []string{http.MethodPost},
		handler: serviceLfsBatch,
	},
	{
		// Git LFS basic object handler
		pattern: serviceLfsBasicMatcher,
		method:  []string{http.MethodGet, http.MethodPut},
		handler: serviceLfsBasic,
	},
	{
		pattern: serviceLfsBasicVerifyMatcher,
		method:  []string{http.MethodPost},
		handler: serviceLfsBasicVerify,
	},
	// Git LFS locks
	{
		pattern: regexp.MustCompile(`(.*?)/info/lfs/locks$`),
		method:  []string{http.MethodPost, http.MethodGet},
		handler: serviceLfsLocks,
	},
	{
		pattern: regexp.MustCompile(`(.*?)/info/lfs/locks/verify$`),
		method:  []string{http.MethodPost},
		handler: serviceLfsLocksVerify,
	},
	{
		pattern: regexp.MustCompile(`(.*?)/info/lfs/locks/([0-9]+)/unlock$`),
		method:  []string{http.MethodPost},
		handler: serviceLfsLocksDelete,
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
		logger := log.FromContext(ctx)
		be := backend.FromContext(ctx)

		// Store repository in context
		// We're not checking for errors here because we want to allow
		// repo creation on the fly.
		repoName := pat.Param(r, "repo")
		repo, _ := be.Repository(ctx, repoName)
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

		if user == nil && !be.AllowKeyless(ctx) {
			askCredentials(w, r)
			renderUnauthorized(w)
			return
		}

		// Store user in context
		ctx = proto.WithUserContext(ctx, user)
		r = r.WithContext(ctx)

		if user != nil {
			logger.Info("found user", "username", user.Username())
		}

		service := git.Service(pat.Param(r, "service"))
		if service == "" {
			// Get service from request params
			service = getServiceType(r)
		}

		accessLevel := be.AccessLevelForUser(ctx, repoName, user)
		ctx = access.WithContext(ctx, accessLevel)
		r = r.WithContext(ctx)

		logger.Info("access level", "repo", repoName, "level", accessLevel)

		file := pat.Param(r, "file")

		// We only allow these services to proceed any other services should return 403
		// - git-upload-pack
		// - git-receive-pack
		// - git-lfs
		switch service {
		case git.UploadPackService:
		case git.ReceivePackService:
			if accessLevel < access.ReadWriteAccess {
				askCredentials(w, r)
				renderUnauthorized(w)
				return
			}

			// Create the repo if it doesn't exist.
			if repo == nil {
				repo, err = be.CreateRepository(ctx, repoName, proto.RepositoryOptions{})
				if err != nil {
					logger.Error("failed to create repository", "repo", repoName, "err", err)
					renderInternalServerError(w)
					return
				}
			}
		case gitLfsService:
			switch {
			case strings.HasPrefix(file, "info/lfs/locks"):
				switch {
				case strings.HasSuffix(file, "lfs/locks"), strings.HasSuffix(file, "/unlock") && r.Method == http.MethodPost:
					// Create lock, list locks, and delete lock require write access
					fallthrough
				case strings.HasSuffix(file, "lfs/locks/verify"):
					// Locks verify requires write access
					// https://github.com/git-lfs/git-lfs/blob/main/docs/api/locking.md#unauthorized-response-2
					if accessLevel < access.ReadWriteAccess {
						renderJSON(w, http.StatusForbidden, lfs.ErrorResponse{
							Message: "write access required",
						})
						return
					}
				}
			case strings.HasPrefix(file, "info/lfs/objects/basic"):
				switch r.Method {
				case http.MethodPut:
					// Basic upload
					if accessLevel < access.ReadWriteAccess {
						renderJSON(w, http.StatusForbidden, lfs.ErrorResponse{
							Message: "write access required",
						})
						return
					}
				case http.MethodGet:
					// Basic download
				case http.MethodPost:
					// Basic verify
				}
			}
			if accessLevel < access.ReadOnlyAccess {
				askCredentials(w, r)
				renderJSON(w, http.StatusUnauthorized, lfs.ErrorResponse{
					Message: "credentials needed",
				})
				return
			}
		default:
			renderForbidden(w)
			return
		}

		// If the repo doesn't exist, return 404
		if repo == nil {
			renderNotFound(w)
			return
		}

		if accessLevel < access.ReadOnlyAccess {
			askCredentials(w, r)
			renderUnauthorized(w)
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
	service, dir, repoName := git.Service(pat.Param(r, "service")), pat.Param(r, "dir"), pat.Param(r, "repo")

	if !isSmart(r, service) {
		renderForbidden(w)
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

	var stdout bytes.Buffer
	cmd := git.ServiceCommand{
		Stdout: &stdout,
		Dir:    dir,
		Args:   []string{"--stateless-rpc"},
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

	// Handle gzip encoding
	reader := r.Body
	defer reader.Close() // nolint: errcheck
	switch r.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err := gzip.NewReader(reader)
		if err != nil {
			logger.Errorf("failed to create gzip reader: %v", err)
			renderInternalServerError(w)
			return
		}
		defer reader.Close() // nolint: errcheck
	}

	cmd.Stdin = reader

	if err := service.Handler(ctx, cmd); err != nil {
		if errors.Is(err, git.ErrInvalidRepo) {
			renderNotFound(w)
			return
		}
		renderInternalServerError(w)
		return
	}

	// Handle buffered output
	// Useful when using proxies

	// We know that `w` is an `http.ResponseWriter`.
	flusher, ok := w.(http.Flusher)
	if !ok {
		logger.Errorf("expected http.ResponseWriter to be an http.Flusher, got %T", w)
		return
	}

	p := make([]byte, 1024)
	for {
		nRead, err := stdout.Read(p)
		if err == io.EOF {
			break
		}
		nWrite, err := w.Write(p[:nRead])
		if err != nil {
			logger.Errorf("failed to write data: %v", err)
			return
		}
		if nRead != nWrite {
			logger.Errorf("failed to write data: %d read, %d written", nRead, nWrite)
			return
		}
		flusher.Flush()
	}

	if service == git.ReceivePackService {
		if err := git.EnsureDefaultBranch(ctx, cmd); err != nil {
			logger.Errorf("failed to ensure default branch: %s", err)
		}
	}
}

func getInfoRefs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cfg := config.FromContext(ctx)
	dir, repoName, file := pat.Param(r, "dir"), pat.Param(r, "repo"), pat.Param(r, "file")
	service := getServiceType(r)
	version := r.Header.Get("Git-Protocol")

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
		if len(version) != 0 {
			cmd.Env = append(cmd.Env, fmt.Sprintf("GIT_PROTOCOL=%s", version))
		}

		if err := service.Handler(ctx, cmd); err != nil {
			renderNotFound(w)
			return
		}

		hdrNocache(w)
		w.Header().Set("Content-Type", fmt.Sprintf("application/x-%s-advertisement", service))
		w.WriteHeader(http.StatusOK)
		if len(version) == 0 {
			git.WritePktline(w, "# service="+service.String()) // nolint: errcheck
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
	contentType := r.Header.Get("Content-Type")
	return strings.HasPrefix(contentType, fmt.Sprintf("application/x-%s-request", service))
}

func updateServerInfo(ctx context.Context, dir string) error {
	return gitb.UpdateServerInfo(ctx, dir)
}

// HTTP error response handling functions

func renderBadRequest(w http.ResponseWriter) {
	renderStatus(http.StatusBadRequest)(w, nil)
}

func renderMethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	if r.Proto == "HTTP/1.1" {
		renderStatus(http.StatusMethodNotAllowed)(w, r)
	} else {
		renderBadRequest(w)
	}
}

func renderNotFound(w http.ResponseWriter) {
	renderStatus(http.StatusNotFound)(w, nil)
}

func renderUnauthorized(w http.ResponseWriter) {
	renderStatus(http.StatusUnauthorized)(w, nil)
}

func renderForbidden(w http.ResponseWriter) {
	renderStatus(http.StatusForbidden)(w, nil)
}

func renderInternalServerError(w http.ResponseWriter) {
	renderStatus(http.StatusInternalServerError)(w, nil)
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
