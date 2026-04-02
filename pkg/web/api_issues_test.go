package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/migrate"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/store"
	"github.com/charmbracelet/soft-serve/pkg/store/database"
	"github.com/gorilla/mux"
	_ "modernc.org/sqlite"
)

// issueAPITestEnv holds all the pieces needed for issue API integration tests.
type issueAPITestEnv struct {
	ctx    context.Context
	be     *backend.Backend
	admin  proto.User
	router http.Handler
	repo   proto.Repository
}

// setupIssueAPIEnv creates a full in-memory backend and registers the
// IssueAPIController onto a fresh mux.Router.
func setupIssueAPIEnv(t *testing.T) issueAPITestEnv {
	t.Helper()

	dp := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.DataPath = dp
	cfg.DB.Driver = "sqlite"
	cfg.DB.DataSource = dp + "/test.db?_pragma=foreign_keys(1)"

	ctx := context.Background()
	ctx = config.WithContext(ctx, cfg)

	dbx, err := db.Open(ctx, cfg.DB.Driver, cfg.DB.DataSource)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { dbx.Close() })

	if err := migrate.Migrate(ctx, dbx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	dbstore := database.New(ctx, dbx)
	ctx = store.WithContext(ctx, dbstore)
	ctx = db.WithContext(ctx, dbx)
	be := backend.New(ctx, cfg, dbx, dbstore)
	ctx = backend.WithContext(ctx, be)

	admin, err := be.User(ctx, "admin")
	if err != nil {
		t.Fatalf("get admin: %v", err)
	}

	repo, err := be.CreateRepository(ctx, "testrepo", admin, proto.RepositoryOptions{})
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}

	r := mux.NewRouter()
	IssueAPIController(ctx, r)

	// Wrap with context middleware so handlers can pull be/store/db.
	h := NewContextHandler(ctx)(r)

	return issueAPITestEnv{
		ctx:    ctx,
		be:     be,
		admin:  admin,
		router: h,
		repo:   repo,
	}
}

// adminToken creates an access token for the admin user and returns the token string.
func adminToken(t *testing.T, env issueAPITestEnv) string {
	t.Helper()
	tok, err := env.be.CreateAccessToken(env.ctx, env.admin, "test-token", time.Time{})
	if err != nil {
		t.Fatalf("create access token: %v", err)
	}
	return tok
}

// doRequest performs an HTTP request against the test server and returns
// the recorded response.
func doRequest(t *testing.T, env issueAPITestEnv, method, path string, body interface{}, authToken string) *httptest.ResponseRecorder {
	t.Helper()

	var reqBody *bytes.Buffer
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = &bytes.Buffer{}
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	if authToken != "" {
		req.Header.Set("Authorization", "Token "+authToken)
	}

	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)
	return rec
}

func TestIssueAPI_ListIssues_EmptyReturns200(t *testing.T) {
	env := setupIssueAPIEnv(t)

	rec := doRequest(t, env, "GET", "/api/v1/repos/testrepo/issues", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var issues []issueResponse
	if err := json.NewDecoder(rec.Body).Decode(&issues); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(issues) != 0 {
		t.Errorf("expected empty list, got %d", len(issues))
	}
}

func TestIssueAPI_CreateIssue_Returns201(t *testing.T) {
	env := setupIssueAPIEnv(t)
	tok := adminToken(t, env)

	body := map[string]string{"title": "Test Issue", "body": "Hello world"}
	rec := doRequest(t, env, "POST", "/api/v1/repos/testrepo/issues", body, tok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var issue issueResponse
	if err := json.NewDecoder(rec.Body).Decode(&issue); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if issue.Title != "Test Issue" {
		t.Errorf("expected title 'Test Issue', got %q", issue.Title)
	}
	if issue.Status != "open" {
		t.Errorf("expected status 'open', got %q", issue.Status)
	}
	if issue.ID == 0 {
		t.Error("expected non-zero ID")
	}
}

func TestIssueAPI_GetIssue_NotFound(t *testing.T) {
	env := setupIssueAPIEnv(t)

	rec := doRequest(t, env, "GET", "/api/v1/repos/testrepo/issues/9999", nil, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}

	var errResp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if errResp["error"] == "" {
		t.Error("expected error message")
	}
}

func TestIssueAPI_GetIssue_Found(t *testing.T) {
	env := setupIssueAPIEnv(t)
	tok := adminToken(t, env)

	// Create an issue first
	body := map[string]string{"title": "My Issue"}
	createRec := doRequest(t, env, "POST", "/api/v1/repos/testrepo/issues", body, tok)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create issue: %d: %s", createRec.Code, createRec.Body.String())
	}
	var created issueResponse
	json.NewDecoder(createRec.Body).Decode(&created) //nolint:errcheck

	path := fmt.Sprintf("/api/v1/repos/testrepo/issues/%d", created.ID)
	rec := doRequest(t, env, "GET", path, nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var issue issueResponse
	if err := json.NewDecoder(rec.Body).Decode(&issue); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if issue.ID != created.ID {
		t.Errorf("expected ID %d, got %d", created.ID, issue.ID)
	}
}

func TestIssueAPI_UpdateIssue_ForbiddenForWrongUser(t *testing.T) {
	env := setupIssueAPIEnv(t)
	adminTok := adminToken(t, env)

	// Create an issue as admin
	body := map[string]string{"title": "Admin Issue"}
	createRec := doRequest(t, env, "POST", "/api/v1/repos/testrepo/issues", body, adminTok)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create issue: %d", createRec.Code)
	}
	var created issueResponse
	json.NewDecoder(createRec.Body).Decode(&created) //nolint:errcheck

	// Create a second user (non-admin)
	otherUser, err := env.be.CreateUser(env.ctx, "otheruser", proto.UserOptions{})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	otherTok, err := env.be.CreateAccessToken(env.ctx, otherUser, "other-token", time.Time{})
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	// Try to update the issue as the other user — should get 403
	path := fmt.Sprintf("/api/v1/repos/testrepo/issues/%d", created.ID)
	updateBody := map[string]string{"title": "Hacked Title"}
	rec := doRequest(t, env, "PATCH", path, updateBody, otherTok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestIssueAPI_ListLabels_Returns200(t *testing.T) {
	env := setupIssueAPIEnv(t)

	rec := doRequest(t, env, "GET", "/api/v1/repos/testrepo/labels", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var labels []labelResponse
	if err := json.NewDecoder(rec.Body).Decode(&labels); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Labels list may be empty; just check it's a valid array
	if labels == nil {
		t.Error("expected non-nil slice")
	}
}

func TestIssueAPI_CreateMilestone_Returns201(t *testing.T) {
	env := setupIssueAPIEnv(t)
	tok := adminToken(t, env)

	body := map[string]string{"title": "v1.0", "description": "First milestone"}
	rec := doRequest(t, env, "POST", "/api/v1/repos/testrepo/milestones", body, tok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var ms milestoneResponse
	if err := json.NewDecoder(rec.Body).Decode(&ms); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if ms.Title != "v1.0" {
		t.Errorf("expected title 'v1.0', got %q", ms.Title)
	}
	if ms.ID == 0 {
		t.Error("expected non-zero ID")
	}
}

func TestIssueAPI_CreateIssue_Unauthenticated(t *testing.T) {
	env := setupIssueAPIEnv(t)

	body := map[string]string{"title": "Test"}
	rec := doRequest(t, env, "POST", "/api/v1/repos/testrepo/issues", body, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestIssueAPI_RepoNotFound_Returns404(t *testing.T) {
	env := setupIssueAPIEnv(t)

	rec := doRequest(t, env, "GET", "/api/v1/repos/nosuchrepo/issues", nil, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestIssueAPI_CreateAndListIssues(t *testing.T) {
	env := setupIssueAPIEnv(t)
	tok := adminToken(t, env)

	// Create 3 issues
	for i := range 3 {
		body := map[string]string{"title": fmt.Sprintf("Issue %d", i+1)}
		rec := doRequest(t, env, "POST", "/api/v1/repos/testrepo/issues", body, tok)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create issue %d: %d: %s", i+1, rec.Code, rec.Body.String())
		}
	}

	rec := doRequest(t, env, "GET", "/api/v1/repos/testrepo/issues", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list: %d: %s", rec.Code, rec.Body.String())
	}

	var issues []issueResponse
	json.NewDecoder(rec.Body).Decode(&issues) //nolint:errcheck
	if len(issues) != 3 {
		t.Errorf("expected 3 issues, got %d", len(issues))
	}
}

func TestIssueAPI_ContentTypeIsJSON(t *testing.T) {
	env := setupIssueAPIEnv(t)

	rec := doRequest(t, env, "GET", "/api/v1/repos/testrepo/issues", nil, "")
	ct := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}
}
