package backend

import (
	"context"
	"testing"

	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/migrate"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/store"
	"github.com/charmbracelet/soft-serve/pkg/store/database"
	"github.com/matryer/is"
	_ "modernc.org/sqlite"
)

type issueTestEnv struct {
	ctx   context.Context
	be    *Backend
	admin proto.User
}

func setupIssueBackend(t *testing.T) issueTestEnv {
	t.Helper()
	is := is.New(t)

	dp := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.DataPath = dp
	cfg.DB.Driver = "sqlite"
	cfg.DB.DataSource = dp + "/test.db?_pragma=foreign_keys(1)"

	ctx := context.Background()
	ctx = config.WithContext(ctx, cfg)

	dbx, err := db.Open(ctx, cfg.DB.Driver, cfg.DB.DataSource)
	is.NoErr(err)
	t.Cleanup(func() { dbx.Close() })

	is.NoErr(migrate.Migrate(ctx, dbx))

	dbstore := database.New(ctx, dbx)
	ctx = store.WithContext(ctx, dbstore)
	be := New(ctx, cfg, dbx, dbstore)
	ctx = WithContext(ctx, be)

	// The migration creates a default "admin" user; look it up instead of creating a duplicate.
	admin, err := be.User(ctx, "admin")
	is.NoErr(err)

	return issueTestEnv{ctx: ctx, be: be, admin: admin}
}

func TestCreateAndGetIssue(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	issue, err := e.be.CreateIssue(e.ctx, "myrepo", e.admin.ID(), "First issue", "Some body")
	is.NoErr(err)
	is.True(issue.ID() > 0)
	is.Equal(issue.Title(), "First issue")
	is.Equal(issue.Body(), "Some body")
	is.Equal(issue.Status(), "open")
	is.True(issue.IsOpen())
	is.True(!issue.IsClosed())

	fetched, err := e.be.GetIssue(e.ctx, issue.ID())
	is.NoErr(err)
	is.Equal(fetched.ID(), issue.ID())
	is.Equal(fetched.Title(), "First issue")
}

func TestGetIssue_NotFound(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.GetIssue(e.ctx, 9999)
	is.True(err != nil)
}

func TestGetIssuesByRepository(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	_, err = e.be.CreateIssue(e.ctx, "myrepo", e.admin.ID(), "Open issue", "")
	is.NoErr(err)

	issue2, err := e.be.CreateIssue(e.ctx, "myrepo", e.admin.ID(), "To be closed", "")
	is.NoErr(err)
	is.NoErr(e.be.CloseIssue(e.ctx, issue2.ID(), issue2.RepoID(), e.admin.ID()))

	// default (empty) returns all
	all, err := e.be.GetIssuesByRepository(e.ctx, "myrepo", "")
	is.NoErr(err)
	is.Equal(len(all), 2)

	open, err := e.be.GetIssuesByRepository(e.ctx, "myrepo", "open")
	is.NoErr(err)
	is.Equal(len(open), 1)
	is.Equal(open[0].Title(), "Open issue")

	closed, err := e.be.GetIssuesByRepository(e.ctx, "myrepo", "closed")
	is.NoErr(err)
	is.Equal(len(closed), 1)
	is.Equal(closed[0].Title(), "To be closed")
}

func TestGetIssuesByRepository_InvalidStatus(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	_, err = e.be.GetIssuesByRepository(e.ctx, "myrepo", "invalid")
	is.True(err != nil)
}

func TestCloseAndReopenIssue(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	issue, err := e.be.CreateIssue(e.ctx, "myrepo", e.admin.ID(), "Issue", "")
	is.NoErr(err)
	is.True(issue.IsOpen())

	is.NoErr(e.be.CloseIssue(e.ctx, issue.ID(), issue.RepoID(), e.admin.ID()))

	fetched, err := e.be.GetIssue(e.ctx, issue.ID())
	is.NoErr(err)
	is.True(fetched.IsClosed())
	is.True(!fetched.ClosedAt().IsZero())
	is.Equal(fetched.ClosedBy(), e.admin.ID())

	is.NoErr(e.be.ReopenIssue(e.ctx, issue.ID(), issue.RepoID()))

	fetched, err = e.be.GetIssue(e.ctx, issue.ID())
	is.NoErr(err)
	is.True(fetched.IsOpen())
	is.True(fetched.ClosedAt().IsZero())
	is.Equal(fetched.ClosedBy(), int64(0))
}

func TestCloseIssue_WrongRepo(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "repo1", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)
	_, err = e.be.CreateRepository(e.ctx, "repo2", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	issue, err := e.be.CreateIssue(e.ctx, "repo1", e.admin.ID(), "Issue in repo1", "")
	is.NoErr(err)

	repo2, err := e.be.Repository(e.ctx, "repo2")
	is.NoErr(err)

	// Close with wrong repoID — the AND repo_id = ? guard means nothing is mutated.
	is.NoErr(e.be.CloseIssue(e.ctx, issue.ID(), repo2.ID(), e.admin.ID()))

	fetched, err := e.be.GetIssue(e.ctx, issue.ID())
	is.NoErr(err)
	is.True(fetched.IsOpen())
}

func TestUpdateIssue_TitleOnly(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	issue, err := e.be.CreateIssue(e.ctx, "myrepo", e.admin.ID(), "Original", "Original body")
	is.NoErr(err)

	// nil body = preserve existing body
	is.NoErr(e.be.UpdateIssue(e.ctx, issue.ID(), issue.RepoID(), "Updated title", nil))

	fetched, err := e.be.GetIssue(e.ctx, issue.ID())
	is.NoErr(err)
	is.Equal(fetched.Title(), "Updated title")
	is.Equal(fetched.Body(), "Original body")
}

func TestUpdateIssue_Body(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	issue, err := e.be.CreateIssue(e.ctx, "myrepo", e.admin.ID(), "Issue", "Old body")
	is.NoErr(err)

	newBody := "New body"
	is.NoErr(e.be.UpdateIssue(e.ctx, issue.ID(), issue.RepoID(), "Issue", &newBody))

	fetched, err := e.be.GetIssue(e.ctx, issue.ID())
	is.NoErr(err)
	is.Equal(fetched.Body(), "New body")
}

func TestUpdateIssue_WrongRepo(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "repo1", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)
	_, err = e.be.CreateRepository(e.ctx, "repo2", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	issue, err := e.be.CreateIssue(e.ctx, "repo1", e.admin.ID(), "Issue", "Body")
	is.NoErr(err)

	repo2, err := e.be.Repository(e.ctx, "repo2")
	is.NoErr(err)

	newBody := "Injected"
	is.NoErr(e.be.UpdateIssue(e.ctx, issue.ID(), repo2.ID(), "Injected title", &newBody))

	// Issue in repo1 must be unchanged.
	fetched, err := e.be.GetIssue(e.ctx, issue.ID())
	is.NoErr(err)
	is.Equal(fetched.Title(), "Issue")
	is.Equal(fetched.Body(), "Body")
}

func TestDeleteIssue(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	issue, err := e.be.CreateIssue(e.ctx, "myrepo", e.admin.ID(), "To delete", "")
	is.NoErr(err)

	is.NoErr(e.be.DeleteIssue(e.ctx, issue.ID(), issue.RepoID()))

	_, err = e.be.GetIssue(e.ctx, issue.ID())
	is.True(err != nil)
}

func TestDeleteIssue_WrongRepo(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "repo1", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)
	_, err = e.be.CreateRepository(e.ctx, "repo2", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	issue, err := e.be.CreateIssue(e.ctx, "repo1", e.admin.ID(), "Issue", "")
	is.NoErr(err)

	repo2, err := e.be.Repository(e.ctx, "repo2")
	is.NoErr(err)

	// Delete with wrong repoID: must be a no-op.
	is.NoErr(e.be.DeleteIssue(e.ctx, issue.ID(), repo2.ID()))

	fetched, err := e.be.GetIssue(e.ctx, issue.ID())
	is.NoErr(err)
	is.Equal(fetched.Title(), "Issue")
}

func TestCountIssues(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	for range 3 {
		_, err = e.be.CreateIssue(e.ctx, "myrepo", e.admin.ID(), "Issue", "")
		is.NoErr(err)
	}

	issue, err := e.be.CreateIssue(e.ctx, "myrepo", e.admin.ID(), "To close", "")
	is.NoErr(err)
	is.NoErr(e.be.CloseIssue(e.ctx, issue.ID(), issue.RepoID(), e.admin.ID()))

	total, err := e.be.CountIssues(e.ctx, "myrepo", "")
	is.NoErr(err)
	is.Equal(total, int64(4))

	open, err := e.be.CountIssues(e.ctx, "myrepo", "open")
	is.NoErr(err)
	is.Equal(open, int64(3))

	closed, err := e.be.CountIssues(e.ctx, "myrepo", "closed")
	is.NoErr(err)
	is.Equal(closed, int64(1))
}

func TestCountIssues_InvalidStatus(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	_, err = e.be.CountIssues(e.ctx, "myrepo", "OPEN")
	is.True(err != nil)
}

func TestCreateIssue_EmptyTitle(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	_, err = e.be.CreateIssue(e.ctx, "myrepo", e.admin.ID(), "", "body")
	is.True(err != nil)
}

func TestUpdateIssue_EmptyTitle(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	issue, err := e.be.CreateIssue(e.ctx, "myrepo", e.admin.ID(), "Original", "body")
	is.NoErr(err)

	err = e.be.UpdateIssue(e.ctx, issue.ID(), issue.RepoID(), "", nil)
	is.True(err != nil)

	// Title must be unchanged.
	fetched, err := e.be.GetIssue(e.ctx, issue.ID())
	is.NoErr(err)
	is.Equal(fetched.Title(), "Original")
}

func TestUpdateIssue_ExplicitEmptyBody(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	issue, err := e.be.CreateIssue(e.ctx, "myrepo", e.admin.ID(), "Issue", "Old body")
	is.NoErr(err)

	// Pointer to "" explicitly clears the body (distinct from nil which preserves it).
	emptyBody := ""
	is.NoErr(e.be.UpdateIssue(e.ctx, issue.ID(), issue.RepoID(), "Issue", &emptyBody))

	fetched, err := e.be.GetIssue(e.ctx, issue.ID())
	is.NoErr(err)
	is.Equal(fetched.Body(), "")
}

func TestCloseIssue_AlreadyClosed(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	issue, err := e.be.CreateIssue(e.ctx, "myrepo", e.admin.ID(), "Issue", "")
	is.NoErr(err)

	is.NoErr(e.be.CloseIssue(e.ctx, issue.ID(), issue.RepoID(), e.admin.ID()))

	// Backend does not guard double-close (idempotent at this layer).
	// The SSH command layer enforces the guard instead.
	is.NoErr(e.be.CloseIssue(e.ctx, issue.ID(), issue.RepoID(), e.admin.ID()))

	fetched, err := e.be.GetIssue(e.ctx, issue.ID())
	is.NoErr(err)
	is.True(fetched.IsClosed())
}

func TestReopenIssue_AlreadyOpen(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	issue, err := e.be.CreateIssue(e.ctx, "myrepo", e.admin.ID(), "Issue", "")
	is.NoErr(err)
	is.True(issue.IsOpen())

	// Backend does not guard reopen-when-already-open (idempotent at this layer).
	is.NoErr(e.be.ReopenIssue(e.ctx, issue.ID(), issue.RepoID()))

	fetched, err := e.be.GetIssue(e.ctx, issue.ID())
	is.NoErr(err)
	is.True(fetched.IsOpen())
}

func TestGetIssuesByRepository_RepoNotFound(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.GetIssuesByRepository(e.ctx, "nonexistent", "open")
	is.True(err != nil)
}

func TestGetIssuesByRepository_StatusAll(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	_, err = e.be.CreateIssue(e.ctx, "myrepo", e.admin.ID(), "Open issue", "")
	is.NoErr(err)

	issue2, err := e.be.CreateIssue(e.ctx, "myrepo", e.admin.ID(), "Closed issue", "")
	is.NoErr(err)
	is.NoErr(e.be.CloseIssue(e.ctx, issue2.ID(), issue2.RepoID(), e.admin.ID()))

	// "all" and "" must both return both issues.
	all, err := e.be.GetIssuesByRepository(e.ctx, "myrepo", "all")
	is.NoErr(err)
	is.Equal(len(all), 2)

	allEmpty, err := e.be.GetIssuesByRepository(e.ctx, "myrepo", "")
	is.NoErr(err)
	is.Equal(len(allEmpty), 2)
}

func TestCountIssues_StatusAll(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	_, err = e.be.CreateIssue(e.ctx, "myrepo", e.admin.ID(), "Open", "")
	is.NoErr(err)

	issue2, err := e.be.CreateIssue(e.ctx, "myrepo", e.admin.ID(), "Closed", "")
	is.NoErr(err)
	is.NoErr(e.be.CloseIssue(e.ctx, issue2.ID(), issue2.RepoID(), e.admin.ID()))

	// "all" and "" must both return total count.
	countAll, err := e.be.CountIssues(e.ctx, "myrepo", "all")
	is.NoErr(err)
	is.Equal(countAll, int64(2))

	countEmpty, err := e.be.CountIssues(e.ctx, "myrepo", "")
	is.NoErr(err)
	is.Equal(countEmpty, int64(2))
}

func TestIssueIsolation_BetweenRepos(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "repo1", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)
	_, err = e.be.CreateRepository(e.ctx, "repo2", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	_, err = e.be.CreateIssue(e.ctx, "repo1", e.admin.ID(), "repo1 issue", "")
	is.NoErr(err)
	_, err = e.be.CreateIssue(e.ctx, "repo2", e.admin.ID(), "repo2 issue", "")
	is.NoErr(err)

	repo1Issues, err := e.be.GetIssuesByRepository(e.ctx, "repo1", "")
	is.NoErr(err)
	is.Equal(len(repo1Issues), 1)
	is.Equal(repo1Issues[0].Title(), "repo1 issue")

	repo2Issues, err := e.be.GetIssuesByRepository(e.ctx, "repo2", "")
	is.NoErr(err)
	is.Equal(len(repo2Issues), 1)
	is.Equal(repo2Issues[0].Title(), "repo2 issue")
}
