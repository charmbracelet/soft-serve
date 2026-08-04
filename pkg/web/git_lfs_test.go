package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/migrate"
	"github.com/charmbracelet/soft-serve/pkg/lfs"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/store"
	"github.com/charmbracelet/soft-serve/pkg/store/database"
	"github.com/gorilla/mux"
	"github.com/matryer/is"
	_ "modernc.org/sqlite"
)

// newLFSTestContext returns a context wired with a config and a real migrated
// SQLite-backed backend, plus the backend and datastore so tests can create
// users, repositories, and locks.
func newLFSTestContext(t *testing.T) (context.Context, *backend.Backend, store.Store) {
	t.Helper()
	is := is.New(t)
	ctx := context.Background()

	dp := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.DataPath = dp
	cfg.DB.Driver = "sqlite"
	cfg.DB.DataSource = dp + "/test.db"

	ctx = config.WithContext(ctx, cfg)
	dbx, err := db.Open(ctx, cfg.DB.Driver, cfg.DB.DataSource)
	is.NoErr(err)
	t.Cleanup(func() { dbx.Close() }) //nolint:errcheck

	is.NoErr(migrate.Migrate(ctx, dbx))
	ctx = db.WithContext(ctx, dbx)
	datastore := database.New(ctx, dbx)
	ctx = store.WithContext(ctx, datastore)
	be := backend.New(ctx, cfg, dbx, datastore)
	ctx = backend.WithContext(ctx, be)

	return ctx, be, datastore
}

// TestLFSLockLookupIsScopedToRepository verifies that a lock cannot be read by
// global lock ID from a repository it does not belong to.
//
// The LFS lock routes authorize the caller against the repository in the URL,
// but the lock list handler accepts an `id` query parameter and used to fetch
// the lock by global ID alone. A user with write access to any repository could
// walk lock IDs and read file paths, owner usernames, and timestamps out of
// private repositories they have no access to.
func TestLFSLockLookupIsScopedToRepository(t *testing.T) {
	is := is.New(t)
	ctx, be, datastore := newLFSTestContext(t)
	dbx := db.FromContext(ctx)

	// The victim owns a private repository with a lock on a sensitive path.
	victim, err := be.CreateUser(ctx, "victim", proto.UserOptions{})
	is.NoErr(err)
	victimRepo, err := be.CreateRepository(ctx, "victim-repo", victim, proto.RepositoryOptions{Private: true})
	is.NoErr(err)
	is.NoErr(datastore.CreateLFSLockForUser(ctx, dbx, victimRepo.ID(), victim.ID(), "secret/plans.bin", "refs/heads/main"))

	victimLock, err := datastore.GetLFSLockForPath(ctx, dbx, victimRepo.ID(), "secret/plans.bin")
	is.NoErr(err)

	// The attacker owns their own repository, so they have write access to
	// it, but no access at all to the victim's repository.
	attacker, err := be.CreateUser(ctx, "attacker", proto.UserOptions{})
	is.NoErr(err)
	attackerRepo, err := be.CreateRepository(ctx, "attacker-repo", attacker, proto.RepositoryOptions{})
	is.NoErr(err)

	// The store must not return a lock that belongs to another repository.
	_, err = datastore.GetLFSLockByID(ctx, dbx, attackerRepo.ID(), victimLock.ID)
	if err == nil {
		t.Fatal("expected error reading another repository's lock by ID")
	}

	// The owner can still read their own lock by ID.
	got, err := datastore.GetLFSLockByID(ctx, dbx, victimRepo.ID(), victimLock.ID)
	is.NoErr(err)
	is.Equal(got.Path, "secret/plans.bin")
}

// TestLFSLocksGetDoesNotLeakAcrossRepositories drives the HTTP lock list
// handler the way an attacker would: authorized for their own repository,
// asking for a lock ID that belongs to somebody else's private repository.
func TestLFSLocksGetDoesNotLeakAcrossRepositories(t *testing.T) {
	is := is.New(t)
	ctx, be, datastore := newLFSTestContext(t)
	dbx := db.FromContext(ctx)

	victim, err := be.CreateUser(ctx, "victim", proto.UserOptions{})
	is.NoErr(err)
	victimRepo, err := be.CreateRepository(ctx, "victim-repo", victim, proto.RepositoryOptions{Private: true})
	is.NoErr(err)
	is.NoErr(datastore.CreateLFSLockForUser(ctx, dbx, victimRepo.ID(), victim.ID(), "secret/plans.bin", "refs/heads/main"))
	victimLock, err := datastore.GetLFSLockForPath(ctx, dbx, victimRepo.ID(), "secret/plans.bin")
	is.NoErr(err)

	attacker, err := be.CreateUser(ctx, "attacker", proto.UserOptions{})
	is.NoErr(err)
	attackerRepo, err := be.CreateRepository(ctx, "attacker-repo", attacker, proto.RepositoryOptions{})
	is.NoErr(err)

	// Request the victim's lock ID while scoped to the attacker's own
	// repository, which is exactly what the route authorizes.
	reqCtx := proto.WithUserContext(ctx, attacker)
	reqCtx = proto.WithRepositoryContext(reqCtx, attackerRepo)
	req := httptest.NewRequestWithContext(reqCtx, http.MethodGet, "/attacker-repo.git/info/lfs/locks?id="+strconv.FormatInt(victimLock.ID, 10), nil)
	req.Header.Set("Accept", lfs.MediaType)

	w := httptest.NewRecorder()
	serviceLfsLocks(w, req)

	body := w.Body.String()
	if strings.Contains(body, "secret/plans.bin") {
		t.Errorf("leaked victim lock path in response: %s", body)
	}
	if strings.Contains(body, "victim") {
		t.Errorf("leaked victim username in response: %s", body)
	}

	// The owner asking for their own lock still gets it, so the fix did not
	// simply break the endpoint.
	ownerCtx := proto.WithUserContext(ctx, victim)
	ownerCtx = proto.WithRepositoryContext(ownerCtx, victimRepo)
	req = httptest.NewRequestWithContext(ownerCtx, http.MethodGet, "/victim-repo.git/info/lfs/locks?id="+strconv.FormatInt(victimLock.ID, 10), nil)
	req.Header.Set("Accept", lfs.MediaType)

	w = httptest.NewRecorder()
	serviceLfsLocks(w, req)
	is.Equal(w.Code, http.StatusOK)

	var resp lfs.LockListResponse
	is.NoErr(json.NewDecoder(w.Body).Decode(&resp))
	is.Equal(len(resp.Locks), 1)
	is.Equal(resp.Locks[0].Path, "secret/plans.bin")
}

// TestLFSLocksDeleteDoesNotLeakAcrossRepositories covers the unlock route,
// which fetched the lock by global ID and echoed its path and owner back in
// the "lock belongs to another user" rejection. Deletion itself was already
// repository-scoped, so this path disclosed metadata rather than destroying
// it, but it is the same unscoped lookup.
func TestLFSLocksDeleteDoesNotLeakAcrossRepositories(t *testing.T) {
	is := is.New(t)
	ctx, be, datastore := newLFSTestContext(t)
	dbx := db.FromContext(ctx)

	victim, err := be.CreateUser(ctx, "victim", proto.UserOptions{})
	is.NoErr(err)
	victimRepo, err := be.CreateRepository(ctx, "victim-repo", victim, proto.RepositoryOptions{Private: true})
	is.NoErr(err)
	is.NoErr(datastore.CreateLFSLockForUser(ctx, dbx, victimRepo.ID(), victim.ID(), "secret/plans.bin", "refs/heads/main"))
	victimLock, err := datastore.GetLFSLockForPath(ctx, dbx, victimRepo.ID(), "secret/plans.bin")
	is.NoErr(err)

	attacker, err := be.CreateUser(ctx, "attacker", proto.UserOptions{})
	is.NoErr(err)
	attackerRepo, err := be.CreateRepository(ctx, "attacker-repo", attacker, proto.RepositoryOptions{})
	is.NoErr(err)

	reqCtx := proto.WithUserContext(ctx, attacker)
	reqCtx = proto.WithRepositoryContext(reqCtx, attackerRepo)
	req := httptest.NewRequestWithContext(reqCtx, http.MethodPost,
		"/attacker-repo.git/info/lfs/locks/"+strconv.FormatInt(victimLock.ID, 10)+"/unlock",
		strings.NewReader(`{"force":false}`))
	req.Header.Set("Accept", lfs.MediaType)
	req.Header.Set("Content-Type", lfs.MediaType)
	req = mux.SetURLVars(req, map[string]string{"lock_id": strconv.FormatInt(victimLock.ID, 10)})

	w := httptest.NewRecorder()
	serviceLfsLocksDelete(w, req)

	// Guard against the request never reaching the lookup: a missing or
	// unparseable lock_id would make this test vacuously pass.
	is.Equal(w.Code, http.StatusNotFound)

	body := w.Body.String()
	if strings.Contains(body, "secret/plans.bin") {
		t.Errorf("leaked victim lock path in unlock response: %s", body)
	}
	if strings.Contains(body, "victim") {
		t.Errorf("leaked victim username in unlock response: %s", body)
	}

	// The victim's lock must still exist.
	still, err := datastore.GetLFSLockByID(ctx, dbx, victimRepo.ID(), victimLock.ID)
	is.NoErr(err)
	is.Equal(still.Path, "secret/plans.bin")
}

// forgeableServices are the service names that select a non-LFS branch in
// withAccess. None of them may change how an LFS route is authorized.
var forgeableServices = []string{"git-upload-pack", "git-receive-pack", "git-upload-archive"}

// lfsTestRouter builds the real git router so requests pass through withParams
// and withAccess rather than reaching a handler directly. The middleware is
// where LFS authorization lives, so a test that calls the handler itself cannot
// see a bypass.
func lfsTestRouter(ctx context.Context) *mux.Router {
	router := mux.NewRouter()
	GitController(ctx, router)
	return router
}

// TestLFSUploadRejectsForgedServiceParam covers an authorization bypass in
// withAccess: the LFS branch was selected by a `service` value that, on an LFS
// route, always came from a caller-supplied query parameter. Because the
// git-upload-pack branch sat earlier in the same switch, appending
// `?service=git-upload-pack` matched that branch instead and skipped every LFS
// write check. Under the shipped defaults (anon-access read-only, LFS enabled)
// an unauthenticated caller could write objects into any public repository.
func TestLFSUploadRejectsForgedServiceParam(t *testing.T) {
	is := is.New(t)
	ctx, be, datastore := newLFSTestContext(t)
	cfg := config.FromContext(ctx)
	dbx := db.FromContext(ctx)

	owner, err := be.CreateUser(ctx, "owner", proto.UserOptions{})
	is.NoErr(err)
	repo, err := be.CreateRepository(ctx, "victim-repo", owner, proto.RepositoryOptions{})
	is.NoErr(err)

	router := lfsTestRouter(ctx)
	oid := strings.Repeat("a", 64)
	path := "/victim-repo.git/info/lfs/objects/basic/" + oid

	upload := func(url string) *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(ctx, http.MethodPut, url, strings.NewReader("payload"))
		req.Header.Set("Content-Type", "application/octet-stream")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}

	// Baseline: anonymous read-only access cannot upload.
	is.Equal(upload(path).Code, http.StatusForbidden)

	for _, service := range forgeableServices {
		w := upload(path + "?service=" + service)
		if w.Code != http.StatusForbidden {
			t.Errorf("service=%s: got status %d, want 403: %s", service, w.Code, w.Body.String())
		}
	}

	// The object must not exist on disk or in the database. The handler writes
	// the body before it parses Content-Length, so an error status alone does
	// not prove nothing was stored.
	objPath := filepath.Join(cfg.DataPath, "lfs", strconv.FormatInt(repo.ID(), 10),
		"objects", oid[0:2], oid[2:4], oid)
	if _, err := os.Stat(objPath); !os.IsNotExist(err) {
		t.Errorf("object written to disk at %s despite rejection", objPath)
	}
	if _, err := datastore.GetLFSObjectByOid(ctx, dbx, repo.ID(), oid); err == nil {
		t.Error("object registered in database despite rejection")
	}
}

// TestLFSLockCreateRejectsForgedServiceParam is the lock-create half of the
// same bypass: a read-only collaborator could take locks on arbitrary paths.
func TestLFSLockCreateRejectsForgedServiceParam(t *testing.T) {
	is := is.New(t)
	ctx, be, datastore := newLFSTestContext(t)
	dbx := db.FromContext(ctx)

	owner, err := be.CreateUser(ctx, "owner", proto.UserOptions{})
	is.NoErr(err)
	repo, err := be.CreateRepository(ctx, "victim-repo", owner, proto.RepositoryOptions{})
	is.NoErr(err)

	// An authenticated user with no more than read access to the repository.
	attacker, err := be.CreateUser(ctx, "attacker", proto.UserOptions{})
	is.NoErr(err)
	token, err := be.CreateAccessToken(ctx, attacker, "test", time.Time{})
	is.NoErr(err)

	router := lfsTestRouter(ctx)

	lock := func(url string) *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(ctx, http.MethodPost, url,
			strings.NewReader(`{"path":"src/critical.bin","ref":{"name":"refs/heads/main"}}`))
		req.Header.Set("Content-Type", lfs.MediaType)
		req.Header.Set("Accept", lfs.MediaType)
		req.Header.Set("Authorization", "token "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}

	is.Equal(lock("/victim-repo.git/info/lfs/locks").Code, http.StatusForbidden)

	for _, service := range forgeableServices {
		w := lock("/victim-repo.git/info/lfs/locks?service=" + service)
		if w.Code != http.StatusForbidden {
			t.Errorf("service=%s: got status %d, want 403: %s", service, w.Code, w.Body.String())
		}
	}

	if _, err := datastore.GetLFSLockForPath(ctx, dbx, repo.ID(), "src/critical.bin"); err == nil {
		t.Error("lock created despite rejection")
	}
}

// TestLFSDisabledRejectsForgedServiceParam checks the administrative kill
// switch. The `lfs.enabled = false` check lived inside the branch the bypass
// skipped, so a forged service value re-enabled LFS on a server where the
// administrator had turned it off.
func TestLFSDisabledRejectsForgedServiceParam(t *testing.T) {
	is := is.New(t)
	ctx, be, _ := newLFSTestContext(t)

	cfg := config.FromContext(ctx)
	cfg.LFS.Enabled = false

	owner, err := be.CreateUser(ctx, "owner", proto.UserOptions{})
	is.NoErr(err)
	_, err = be.CreateRepository(ctx, "victim-repo", owner, proto.RepositoryOptions{})
	is.NoErr(err)

	router := lfsTestRouter(ctx)
	url := "/victim-repo.git/info/lfs/objects/basic/" + strings.Repeat("a", 64)

	for _, service := range forgeableServices {
		req := httptest.NewRequestWithContext(ctx, http.MethodPut, url+"?service="+service,
			strings.NewReader("payload"))
		req.Header.Set("Content-Type", "application/octet-stream")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("service=%s: got status %d, want 404 with LFS disabled: %s", service, w.Code, w.Body.String())
		}
	}
}

// TestLFSWriteStillWorksForAuthorizedUser guards against the fix simply
// breaking LFS: a user with write access must still be able to upload an
// object and take a lock, with or without a service parameter present.
func TestLFSWriteStillWorksForAuthorizedUser(t *testing.T) {
	is := is.New(t)
	ctx, be, datastore := newLFSTestContext(t)
	dbx := db.FromContext(ctx)

	owner, err := be.CreateUser(ctx, "owner", proto.UserOptions{})
	is.NoErr(err)
	repo, err := be.CreateRepository(ctx, "owner-repo", owner, proto.RepositoryOptions{})
	is.NoErr(err)
	token, err := be.CreateAccessToken(ctx, owner, "test", time.Time{})
	is.NoErr(err)

	router := lfsTestRouter(ctx)
	oid := strings.Repeat("b", 64)
	body := "payload"

	req := httptest.NewRequestWithContext(ctx, http.MethodPut,
		"/owner-repo.git/info/lfs/objects/basic/"+oid, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-Length", strconv.Itoa(len(body)))
	req.Header.Set("Authorization", "token "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	is.Equal(w.Code, http.StatusOK)

	obj, err := datastore.GetLFSObjectByOid(ctx, dbx, repo.ID(), oid)
	is.NoErr(err)
	is.Equal(obj.Oid, oid)

	// A service parameter on an LFS route is ignored, not rejected.
	req = httptest.NewRequestWithContext(ctx, http.MethodPost,
		"/owner-repo.git/info/lfs/locks?service=git-upload-pack",
		strings.NewReader(`{"path":"src/file.bin","ref":{"name":"refs/heads/main"}}`))
	req.Header.Set("Content-Type", lfs.MediaType)
	req.Header.Set("Accept", lfs.MediaType)
	req.Header.Set("Authorization", "token "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	is.Equal(w.Code, http.StatusCreated)

	got, err := datastore.GetLFSLockForPath(ctx, dbx, repo.ID(), "src/file.bin")
	is.NoErr(err)
	is.Equal(got.Path, "src/file.bin")
}
