package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

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
