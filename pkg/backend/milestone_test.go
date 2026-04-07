package backend

import (
	"testing"

	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/store"
	"github.com/matryer/is"
)

func TestCreateAndGetMilestone(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "msrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	ms, err := e.be.CreateMilestone(e.ctx, "msrepo", "v1.0", "First release")
	is.NoErr(err)
	is.True(ms.ID() > 0)
	is.Equal(ms.Title(), "v1.0")
	is.Equal(ms.Description(), "First release")
	is.True(ms.IsOpen())
	is.True(!ms.IsClosed())

	fetched, err := e.be.GetMilestone(e.ctx, "msrepo", ms.ID())
	is.NoErr(err)
	is.Equal(fetched.ID(), ms.ID())
	is.Equal(fetched.Title(), "v1.0")
}

func TestCreateMilestone_EmptyTitle(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "msrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	_, err = e.be.CreateMilestone(e.ctx, "msrepo", "", "")
	is.True(err != nil)
}

func TestListMilestones_OpenAndClosed(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "msrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	ms1, err := e.be.CreateMilestone(e.ctx, "msrepo", "v1.0", "")
	is.NoErr(err)
	ms2, err := e.be.CreateMilestone(e.ctx, "msrepo", "v2.0", "")
	is.NoErr(err)

	// Close ms1.
	err = e.be.CloseMilestone(e.ctx, "msrepo", ms1.ID())
	is.NoErr(err)

	open, err := e.be.ListMilestones(e.ctx, "msrepo", true)
	is.NoErr(err)
	is.Equal(len(open), 1)
	is.Equal(open[0].ID(), ms2.ID())

	closed, err := e.be.ListMilestones(e.ctx, "msrepo", false)
	is.NoErr(err)
	is.Equal(len(closed), 1)
	is.Equal(closed[0].ID(), ms1.ID())
}

func TestUpdateMilestone(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "msrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	ms, err := e.be.CreateMilestone(e.ctx, "msrepo", "v1.0", "old desc")
	is.NoErr(err)

	err = e.be.UpdateMilestone(e.ctx, "msrepo", ms.ID(), "v1.1", "new desc")
	is.NoErr(err)

	updated, err := e.be.GetMilestone(e.ctx, "msrepo", ms.ID())
	is.NoErr(err)
	is.Equal(updated.Title(), "v1.1")
	is.Equal(updated.Description(), "new desc")
}

func TestCloseMilestone(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "msrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	ms, err := e.be.CreateMilestone(e.ctx, "msrepo", "v1.0", "")
	is.NoErr(err)
	is.True(ms.IsOpen())

	err = e.be.CloseMilestone(e.ctx, "msrepo", ms.ID())
	is.NoErr(err)

	fetched, err := e.be.GetMilestone(e.ctx, "msrepo", ms.ID())
	is.NoErr(err)
	is.True(fetched.IsClosed())
	is.True(!fetched.ClosedAt().IsZero())
}

func TestReopenMilestone(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "msrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	ms, err := e.be.CreateMilestone(e.ctx, "msrepo", "v1.0", "")
	is.NoErr(err)

	err = e.be.CloseMilestone(e.ctx, "msrepo", ms.ID())
	is.NoErr(err)

	err = e.be.ReopenMilestone(e.ctx, "msrepo", ms.ID())
	is.NoErr(err)

	fetched, err := e.be.GetMilestone(e.ctx, "msrepo", ms.ID())
	is.NoErr(err)
	is.True(fetched.IsOpen())
	is.True(fetched.ClosedAt().IsZero())
}

func TestDeleteMilestone(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "msrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	ms, err := e.be.CreateMilestone(e.ctx, "msrepo", "v1.0", "")
	is.NoErr(err)

	err = e.be.DeleteMilestone(e.ctx, "msrepo", ms.ID())
	is.NoErr(err)

	_, err = e.be.GetMilestone(e.ctx, "msrepo", ms.ID())
	is.True(err != nil)
}

func TestSetAndGetIssueMilestone(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "msrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	issue, err := e.be.CreateIssue(e.ctx, "msrepo", e.admin.ID(), "Issue one", "")
	is.NoErr(err)

	ms, err := e.be.CreateMilestone(e.ctx, "msrepo", "v1.0", "")
	is.NoErr(err)

	err = e.be.SetIssueMilestone(e.ctx, issue.ID(), ms.ID())
	is.NoErr(err)

	got, err := e.be.GetIssueMilestone(e.ctx, issue.ID())
	is.NoErr(err)
	is.True(got != nil)
	is.Equal(got.ID(), ms.ID())
}

func TestUnsetIssueMilestone(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "msrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	issue, err := e.be.CreateIssue(e.ctx, "msrepo", e.admin.ID(), "Issue one", "")
	is.NoErr(err)

	ms, err := e.be.CreateMilestone(e.ctx, "msrepo", "v1.0", "")
	is.NoErr(err)

	err = e.be.SetIssueMilestone(e.ctx, issue.ID(), ms.ID())
	is.NoErr(err)

	err = e.be.UnsetIssueMilestone(e.ctx, issue.ID())
	is.NoErr(err)

	got, err := e.be.GetIssueMilestone(e.ctx, issue.ID())
	is.NoErr(err)
	is.True(got == nil)
}

func TestGetIssuesByMilestone(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "msrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	issue1, err := e.be.CreateIssue(e.ctx, "msrepo", e.admin.ID(), "Issue one", "")
	is.NoErr(err)
	issue2, err := e.be.CreateIssue(e.ctx, "msrepo", e.admin.ID(), "Issue two", "")
	is.NoErr(err)

	ms, err := e.be.CreateMilestone(e.ctx, "msrepo", "v1.0", "")
	is.NoErr(err)

	err = e.be.SetIssueMilestone(e.ctx, issue1.ID(), ms.ID())
	is.NoErr(err)

	issues, err := e.be.GetIssuesByRepository(e.ctx, "msrepo", store.IssueFilter{
		MilestoneID: ms.ID(),
	})
	is.NoErr(err)
	is.Equal(len(issues), 1)
	is.Equal(issues[0].ID(), issue1.ID())
	_ = issue2
}

func TestDeleteMilestone_NullifiesIssues(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "msrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	issue, err := e.be.CreateIssue(e.ctx, "msrepo", e.admin.ID(), "Issue one", "")
	is.NoErr(err)

	ms, err := e.be.CreateMilestone(e.ctx, "msrepo", "v1.0", "")
	is.NoErr(err)

	err = e.be.SetIssueMilestone(e.ctx, issue.ID(), ms.ID())
	is.NoErr(err)

	// Delete milestone — should set milestone_id to NULL on issues.
	err = e.be.DeleteMilestone(e.ctx, "msrepo", ms.ID())
	is.NoErr(err)

	got, err := e.be.GetIssueMilestone(e.ctx, issue.ID())
	is.NoErr(err)
	is.True(got == nil)
}
