package backend

import (
	"testing"

	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/matryer/is"
)

func TestCreateAndGetLabel(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	lbl, err := e.be.CreateLabel(e.ctx, "myrepo", "bug", "#ff0000", "Something is broken")
	is.NoErr(err)
	is.True(lbl.ID() > 0)
	is.Equal(lbl.Name(), "bug")
	is.Equal(lbl.Color(), "#ff0000")
	is.Equal(lbl.Description(), "Something is broken")
	is.True(!lbl.CreatedAt().IsZero())
}

func TestCreateLabel_EmptyName(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	_, err = e.be.CreateLabel(e.ctx, "myrepo", "", "", "")
	is.True(err != nil)
}

func TestCreateLabel_SpaceInName(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	_, err = e.be.CreateLabel(e.ctx, "myrepo", "has space", "", "")
	is.True(err != nil)
}

func TestCreateLabel_DuplicateName(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	_, err = e.be.CreateLabel(e.ctx, "myrepo", "bug", "", "")
	is.NoErr(err)

	_, err = e.be.CreateLabel(e.ctx, "myrepo", "bug", "", "")
	is.True(err != nil)
}

func TestListLabels(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	_, err = e.be.CreateLabel(e.ctx, "myrepo", "enhancement", "", "")
	is.NoErr(err)
	_, err = e.be.CreateLabel(e.ctx, "myrepo", "bug", "", "")
	is.NoErr(err)

	labels, err := e.be.ListLabels(e.ctx, "myrepo")
	is.NoErr(err)
	is.Equal(len(labels), 2)
	// Ordered alphabetically.
	is.Equal(labels[0].Name(), "bug")
	is.Equal(labels[1].Name(), "enhancement")
}

func TestListLabels_Empty(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	labels, err := e.be.ListLabels(e.ctx, "myrepo")
	is.NoErr(err)
	is.Equal(len(labels), 0)
}

func TestUpdateLabel(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	lbl, err := e.be.CreateLabel(e.ctx, "myrepo", "bug", "", "")
	is.NoErr(err)

	err = e.be.UpdateLabel(e.ctx, "myrepo", lbl.ID(), "critical-bug", "#cc0000", "High priority")
	is.NoErr(err)

	updated, err := e.be.GetLabel(e.ctx, "myrepo", "critical-bug")
	is.NoErr(err)
	is.Equal(updated.Name(), "critical-bug")
	is.Equal(updated.Color(), "#cc0000")
	is.Equal(updated.Description(), "High priority")
}

func TestDeleteLabel(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	lbl, err := e.be.CreateLabel(e.ctx, "myrepo", "wontfix", "", "")
	is.NoErr(err)

	err = e.be.DeleteLabel(e.ctx, "myrepo", lbl.ID())
	is.NoErr(err)

	labels, err := e.be.ListLabels(e.ctx, "myrepo")
	is.NoErr(err)
	is.Equal(len(labels), 0)
}

func TestAddAndRemoveLabelFromIssue(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	issue, err := e.be.CreateIssue(e.ctx, "myrepo", e.admin.ID(), "Issue", "")
	is.NoErr(err)

	lbl, err := e.be.CreateLabel(e.ctx, "myrepo", "bug", "", "")
	is.NoErr(err)

	err = e.be.AddLabelToIssue(e.ctx, issue.ID(), lbl.ID())
	is.NoErr(err)

	labels, err := e.be.GetIssueLabels(e.ctx, issue.ID())
	is.NoErr(err)
	is.Equal(len(labels), 1)
	is.Equal(labels[0].Name(), "bug")

	err = e.be.RemoveLabelFromIssue(e.ctx, issue.ID(), lbl.ID())
	is.NoErr(err)

	labels, err = e.be.GetIssueLabels(e.ctx, issue.ID())
	is.NoErr(err)
	is.Equal(len(labels), 0)
}

func TestAddLabelToIssue_Idempotent(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	issue, err := e.be.CreateIssue(e.ctx, "myrepo", e.admin.ID(), "Issue", "")
	is.NoErr(err)

	lbl, err := e.be.CreateLabel(e.ctx, "myrepo", "bug", "", "")
	is.NoErr(err)

	is.NoErr(e.be.AddLabelToIssue(e.ctx, issue.ID(), lbl.ID()))
	is.NoErr(e.be.AddLabelToIssue(e.ctx, issue.ID(), lbl.ID())) // second add is a no-op

	labels, err := e.be.GetIssueLabels(e.ctx, issue.ID())
	is.NoErr(err)
	is.Equal(len(labels), 1)
}

func TestGetIssuesByRepositoryAndLabel(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	issue1, err := e.be.CreateIssue(e.ctx, "myrepo", e.admin.ID(), "Bug issue", "")
	is.NoErr(err)
	issue2, err := e.be.CreateIssue(e.ctx, "myrepo", e.admin.ID(), "Feature issue", "")
	is.NoErr(err)

	bug, err := e.be.CreateLabel(e.ctx, "myrepo", "bug", "", "")
	is.NoErr(err)
	feat, err := e.be.CreateLabel(e.ctx, "myrepo", "feature", "", "")
	is.NoErr(err)

	is.NoErr(e.be.AddLabelToIssue(e.ctx, issue1.ID(), bug.ID()))
	is.NoErr(e.be.AddLabelToIssue(e.ctx, issue2.ID(), feat.ID()))

	bugIssues, err := e.be.GetIssuesByRepositoryAndLabel(e.ctx, "myrepo", "bug", "open")
	is.NoErr(err)
	is.Equal(len(bugIssues), 1)
	is.Equal(bugIssues[0].ID(), issue1.ID())

	featIssues, err := e.be.GetIssuesByRepositoryAndLabel(e.ctx, "myrepo", "feature", "open")
	is.NoErr(err)
	is.Equal(len(featIssues), 1)
	is.Equal(featIssues[0].ID(), issue2.ID())
}

func TestDeleteLabel_CascadesIssueLabelRows(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	issue, err := e.be.CreateIssue(e.ctx, "myrepo", e.admin.ID(), "Issue", "")
	is.NoErr(err)

	lbl, err := e.be.CreateLabel(e.ctx, "myrepo", "temp", "", "")
	is.NoErr(err)

	is.NoErr(e.be.AddLabelToIssue(e.ctx, issue.ID(), lbl.ID()))

	// Deleting the label must remove the issue_labels row via ON DELETE CASCADE.
	is.NoErr(e.be.DeleteLabel(e.ctx, "myrepo", lbl.ID()))

	labels, err := e.be.GetIssueLabels(e.ctx, issue.ID())
	is.NoErr(err)
	is.Equal(len(labels), 0)
}

func TestCreateLabel_NamedColors(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"red", "#ff0000"},
		{"blue", "#0000ff"},
		{"yellow", "#ffff00"},
		{"green", "#008000"},
		{"magenta", "#ff00ff"},
		{"pink", "#ffc0cb"},
		{"white", "#ffffff"},
		{"black", "#000000"},
		{"RED", "#ff0000"},   // case-insensitive
		{"#ff0000", "#ff0000"}, // passthrough
		{"ff0000", "ff0000"},   // bare hex passthrough
		{"", ""},               // empty passthrough
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.input, func(t *testing.T) {
			is := is.New(t)
			e := setupIssueBackend(t)

			_, err := e.be.CreateRepository(e.ctx, "myrepo", e.admin, proto.RepositoryOptions{})
			is.NoErr(err)

			lbl, err := e.be.CreateLabel(e.ctx, "myrepo", "lbl-"+tc.input, tc.input, "")
			is.NoErr(err)
			is.Equal(lbl.Color(), tc.want)
		})
	}
}

func TestLabelIsolation_BetweenRepos(t *testing.T) {
	is := is.New(t)
	e := setupIssueBackend(t)

	_, err := e.be.CreateRepository(e.ctx, "repo1", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)
	_, err = e.be.CreateRepository(e.ctx, "repo2", e.admin, proto.RepositoryOptions{})
	is.NoErr(err)

	_, err = e.be.CreateLabel(e.ctx, "repo1", "bug", "", "")
	is.NoErr(err)

	// The same label name in repo2 is independent.
	_, err = e.be.CreateLabel(e.ctx, "repo2", "bug", "#0000ff", "")
	is.NoErr(err)

	r1Labels, err := e.be.ListLabels(e.ctx, "repo1")
	is.NoErr(err)
	is.Equal(len(r1Labels), 1)

	r2Labels, err := e.be.ListLabels(e.ctx, "repo2")
	is.NoErr(err)
	is.Equal(len(r2Labels), 1)
	is.Equal(r2Labels[0].Color(), "#0000ff")
}
