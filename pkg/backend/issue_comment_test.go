package backend

import (
	"testing"

	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/matryer/is"
)

func TestAddAndGetComment(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	issue, err := e.be.CreateIssue(e.ctx, "myrepo", e.admin.ID(), "Issue", "")
	is.NoErr(err)

	comment, err := e.be.AddIssueComment(e.ctx, issue.ID(), e.admin.ID(), "First comment")
	is.NoErr(err)
	is.True(comment.ID() > 0)
	is.Equal(comment.IssueID(), issue.ID())
	is.Equal(comment.UserID(), e.admin.ID())
	is.Equal(comment.Body(), "First comment")
	is.True(!comment.CreatedAt().IsZero())

	fetched, err := e.be.GetIssueComment(e.ctx, comment.ID())
	is.NoErr(err)
	is.Equal(fetched.ID(), comment.ID())
	is.Equal(fetched.Body(), "First comment")
}

func TestAddComment_EmptyBody(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	issue, err := e.be.CreateIssue(e.ctx, "myrepo", e.admin.ID(), "Issue", "")
	is.NoErr(err)

	_, err = e.be.AddIssueComment(e.ctx, issue.ID(), e.admin.ID(), "")
	is.True(err != nil)
}

func TestAddComment_IssueNotFound(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.AddIssueComment(e.ctx, 9999, e.admin.ID(), "body")
	is.True(err != nil)
}

func TestGetIssueComments(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	issue, err := e.be.CreateIssue(e.ctx, "myrepo", e.admin.ID(), "Issue", "")
	is.NoErr(err)

	_, err = e.be.AddIssueComment(e.ctx, issue.ID(), e.admin.ID(), "First")
	is.NoErr(err)
	_, err = e.be.AddIssueComment(e.ctx, issue.ID(), e.admin.ID(), "Second")
	is.NoErr(err)

	comments, err := e.be.GetIssueComments(e.ctx, issue.ID())
	is.NoErr(err)
	is.Equal(len(comments), 2)
	// Comments must be ordered oldest first.
	is.Equal(comments[0].Body(), "First")
	is.Equal(comments[1].Body(), "Second")
}

func TestGetIssueComments_Empty(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	issue, err := e.be.CreateIssue(e.ctx, "myrepo", e.admin.ID(), "Issue", "")
	is.NoErr(err)

	comments, err := e.be.GetIssueComments(e.ctx, issue.ID())
	is.NoErr(err)
	is.Equal(len(comments), 0)
}

func TestGetIssueComments_IssueNotFound(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.GetIssueComments(e.ctx, 9999)
	is.True(err != nil)
}

func TestUpdateIssueComment(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	issue, err := e.be.CreateIssue(e.ctx, "myrepo", e.admin.ID(), "Issue", "")
	is.NoErr(err)

	comment, err := e.be.AddIssueComment(e.ctx, issue.ID(), e.admin.ID(), "Original")
	is.NoErr(err)

	is.NoErr(e.be.UpdateIssueComment(e.ctx, comment.ID(), "Updated"))

	fetched, err := e.be.GetIssueComment(e.ctx, comment.ID())
	is.NoErr(err)
	is.Equal(fetched.Body(), "Updated")
	is.True(fetched.UpdatedAt().After(fetched.CreatedAt()) || fetched.UpdatedAt().Equal(fetched.CreatedAt()))
}

func TestUpdateIssueComment_EmptyBody(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	issue, err := e.be.CreateIssue(e.ctx, "myrepo", e.admin.ID(), "Issue", "")
	is.NoErr(err)

	comment, err := e.be.AddIssueComment(e.ctx, issue.ID(), e.admin.ID(), "Original")
	is.NoErr(err)

	err = e.be.UpdateIssueComment(e.ctx, comment.ID(), "")
	is.True(err != nil)

	// Body must be unchanged.
	fetched, err := e.be.GetIssueComment(e.ctx, comment.ID())
	is.NoErr(err)
	is.Equal(fetched.Body(), "Original")
}

func TestDeleteIssueComment(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	issue, err := e.be.CreateIssue(e.ctx, "myrepo", e.admin.ID(), "Issue", "")
	is.NoErr(err)

	comment, err := e.be.AddIssueComment(e.ctx, issue.ID(), e.admin.ID(), "To delete")
	is.NoErr(err)

	is.NoErr(e.be.DeleteIssueComment(e.ctx, comment.ID()))

	_, err = e.be.GetIssueComment(e.ctx, comment.ID())
	is.True(err != nil)
}

func TestDeleteIssue_CascadesComments(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	issue, err := e.be.CreateIssue(e.ctx, "myrepo", e.admin.ID(), "Issue", "")
	is.NoErr(err)

	comment, err := e.be.AddIssueComment(e.ctx, issue.ID(), e.admin.ID(), "Orphan?")
	is.NoErr(err)

	// Deleting the issue must cascade-delete its comments (ON DELETE CASCADE).
	is.NoErr(e.be.DeleteIssue(e.ctx, issue.ID(), issue.RepoID()))

	_, err = e.be.GetIssueComment(e.ctx, comment.ID())
	is.True(err != nil)
}

func TestCommentIsolation_BetweenIssues(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	issue1, err := e.be.CreateIssue(e.ctx, "myrepo", e.admin.ID(), "Issue 1", "")
	is.NoErr(err)
	issue2, err := e.be.CreateIssue(e.ctx, "myrepo", e.admin.ID(), "Issue 2", "")
	is.NoErr(err)

	_, err = e.be.AddIssueComment(e.ctx, issue1.ID(), e.admin.ID(), "Comment on issue 1")
	is.NoErr(err)
	_, err = e.be.AddIssueComment(e.ctx, issue2.ID(), e.admin.ID(), "Comment on issue 2")
	is.NoErr(err)

	c1, err := e.be.GetIssueComments(e.ctx, issue1.ID())
	is.NoErr(err)
	is.Equal(len(c1), 1)
	is.Equal(c1[0].Body(), "Comment on issue 1")

	c2, err := e.be.GetIssueComments(e.ctx, issue2.ID())
	is.NoErr(err)
	is.Equal(len(c2), 1)
	is.Equal(c2[0].Body(), "Comment on issue 2")
}
