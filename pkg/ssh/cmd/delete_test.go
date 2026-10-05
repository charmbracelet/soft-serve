package cmd

import (
	"errors"
	"testing"

	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/matryer/is"
)

// TestDeleteRequiresRepoAdmin is the regression test for the privilege
// escalation where any read-write collaborator could delete the whole
// repository. Only a repo admin (or a global admin) should be able to.
func TestDeleteRequiresRepoAdmin(t *testing.T) {
	is := is.New(t)
	ctx, be := newAuthTestContext(t)

	ownerCtx := withUser(t, ctx, be, "owner", false)
	owner := proto.UserFromContext(ownerCtx)
	_, err := be.CreateRepository(ownerCtx, "repo", owner, proto.RepositoryOptions{})
	is.NoErr(err)

	collabCtx := withUser(t, ctx, be, "collab", false)
	is.NoErr(be.AddCollaborator(ownerCtx, "repo", "collab", access.ReadWriteAccess))
	is.Equal(be.AccessLevelForUser(collabCtx, "repo", proto.UserFromContext(collabCtx)), access.ReadWriteAccess)

	// A read-write collaborator must not be able to delete the repo.
	err = runRepo(t, collabCtx, "delete", "repo")
	if !errors.Is(err, proto.ErrUnauthorized) {
		t.Fatalf("collab delete: expected ErrUnauthorized, got %v", err)
	}
	_, err = be.Repository(ctx, "repo")
	is.NoErr(err) // repo must still exist

	// An admin-access collaborator can delete the repo.
	adminCtx := withUser(t, ctx, be, "admin-collab", false)
	is.NoErr(be.AddCollaborator(ownerCtx, "repo", "admin-collab", access.AdminAccess))
	is.NoErr(runRepo(t, adminCtx, "delete", "repo"))
	_, err = be.Repository(ctx, "repo")
	if err == nil {
		t.Fatal("expected repo to be deleted")
	}
}
