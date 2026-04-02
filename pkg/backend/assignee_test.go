package backend

import (
	"testing"

	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/matryer/is"
)

func TestAssignAndGetAssignees(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	issue, err := e.be.CreateIssue(e.ctx, "myrepo", e.admin.ID(), "Test issue", "")
	is.NoErr(err)

	is.NoErr(e.be.AssignUserToIssue(e.ctx, "myrepo", issue.ID(), "admin"))

	assignees, err := e.be.GetIssueAssignees(e.ctx, issue.ID())
	is.NoErr(err)
	is.Equal(len(assignees), 1)
	is.Equal(assignees[0].Username(), "admin")
}

func TestAssignUser_Idempotent(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	issue, err := e.be.CreateIssue(e.ctx, "myrepo", e.admin.ID(), "Test issue", "")
	is.NoErr(err)

	// Assign twice — must be idempotent (no error).
	is.NoErr(e.be.AssignUserToIssue(e.ctx, "myrepo", issue.ID(), "admin"))
	is.NoErr(e.be.AssignUserToIssue(e.ctx, "myrepo", issue.ID(), "admin"))

	assignees, err := e.be.GetIssueAssignees(e.ctx, issue.ID())
	is.NoErr(err)
	is.Equal(len(assignees), 1)
}

func TestUnassignUser(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	issue, err := e.be.CreateIssue(e.ctx, "myrepo", e.admin.ID(), "Test issue", "")
	is.NoErr(err)

	is.NoErr(e.be.AssignUserToIssue(e.ctx, "myrepo", issue.ID(), "admin"))

	assignees, err := e.be.GetIssueAssignees(e.ctx, issue.ID())
	is.NoErr(err)
	is.Equal(len(assignees), 1)

	is.NoErr(e.be.UnassignUserFromIssue(e.ctx, "myrepo", issue.ID(), "admin"))

	assignees, err = e.be.GetIssueAssignees(e.ctx, issue.ID())
	is.NoErr(err)
	is.Equal(len(assignees), 0)
}

func TestAssignUser_UserNotFound(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	issue, err := e.be.CreateIssue(e.ctx, "myrepo", e.admin.ID(), "Test issue", "")
	is.NoErr(err)

	err = e.be.AssignUserToIssue(e.ctx, "myrepo", issue.ID(), "nonexistentuser")
	is.True(err != nil)
}

func TestDeleteIssue_CascadesAssignees(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	issue, err := e.be.CreateIssue(e.ctx, "myrepo", e.admin.ID(), "Test issue", "")
	is.NoErr(err)

	is.NoErr(e.be.AssignUserToIssue(e.ctx, "myrepo", issue.ID(), "admin"))

	assignees, err := e.be.GetIssueAssignees(e.ctx, issue.ID())
	is.NoErr(err)
	is.Equal(len(assignees), 1)

	// Delete the issue — the issue_assignees row must cascade away.
	is.NoErr(e.be.DeleteIssue(e.ctx, issue.ID(), issue.RepoID()))

	// After deletion the issue is gone; assignees query returns nothing.
	assignees, err = e.be.GetIssueAssignees(e.ctx, issue.ID())
	is.NoErr(err)
	is.Equal(len(assignees), 0)
}
