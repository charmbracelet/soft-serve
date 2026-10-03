package backend

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/store"
	"github.com/matryer/is"
)

// TestAccessLevelCollaboratorPrivateRepo verifies that the anonymous access
// floor does not raise a collaborator's explicit level on a private
// repository, while still applying on public repositories.
func TestAccessLevelCollaboratorPrivateRepo(t *testing.T) {
	tests := []struct {
		name    string
		private bool
		anon    access.AccessLevel
		collab  access.AccessLevel
		want    access.AccessLevel
	}{
		{"private no-access collab, anon read-only", true, access.ReadOnlyAccess, access.NoAccess, access.NoAccess},
		{"private read-only collab, anon read-write", true, access.ReadWriteAccess, access.ReadOnlyAccess, access.ReadOnlyAccess},
		{"private read-write collab, anon read-only", true, access.ReadOnlyAccess, access.ReadWriteAccess, access.ReadWriteAccess},
		{"public no-access collab, anon read-only", false, access.ReadOnlyAccess, access.NoAccess, access.ReadOnlyAccess},
		{"public read-only collab, anon read-write", false, access.ReadWriteAccess, access.ReadOnlyAccess, access.ReadWriteAccess},
		{"public no-access collab, anon no-access", false, access.NoAccess, access.NoAccess, access.NoAccess},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			be, _ := newTestBackend(t)
			ctx := context.Background()

			is.NoErr(be.SetAnonAccess(ctx, tt.anon))

			_, err := be.CreateRepository(ctx, "repo", nil, proto.RepositoryOptions{Private: tt.private})
			is.NoErr(err)

			user, err := be.CreateUser(ctx, "collab", proto.UserOptions{})
			is.NoErr(err)
			is.NoErr(be.db.TransactionContext(ctx, func(tx *db.Tx) error {
				return be.store.AddCollabByUsernameAndRepo(ctx, tx, "collab", "repo", tt.collab)
			}))

			is.Equal(be.AccessLevelForUser(ctx, "repo", user), tt.want)
		})
	}
}

// TestAccessLevelPrivateRepoNonCollaborator verifies that a signed-in user
// who is not a collaborator has no access to a private repository.
func TestAccessLevelPrivateRepoNonCollaborator(t *testing.T) {
	is := is.New(t)
	be, _ := newTestBackend(t)
	ctx := context.Background()

	_, err := be.CreateRepository(ctx, "repo", nil, proto.RepositoryOptions{Private: true})
	is.NoErr(err)

	user, err := be.CreateUser(ctx, "stranger", proto.UserOptions{})
	is.NoErr(err)

	is.Equal(be.AccessLevelForUser(ctx, "repo", user), access.NoAccess)
	is.Equal(be.AccessLevelForUser(ctx, "repo", nil), access.NoAccess)
}

// TestDeleteUserWithRepositories verifies that a user who owns
// repositories is not deleted until those repositories are gone, and that
// their repositories are left untouched by the refusal.
func TestDeleteUserWithRepositories(t *testing.T) {
	is := is.New(t)
	be, cfg := newTestBackend(t)

	ctx := context.Background()
	ctx = config.WithContext(ctx, cfg)
	ctx = db.WithContext(ctx, be.db)
	ctx = store.WithContext(ctx, be.store)
	admin, err := be.User(ctx, "admin")
	is.NoErr(err)
	ctx = proto.WithUserContext(ctx, admin)

	owner, err := be.CreateUser(ctx, "owner", proto.UserOptions{})
	is.NoErr(err)
	_, err = be.CreateRepository(ctx, "owned", owner, proto.RepositoryOptions{})
	is.NoErr(err)

	err = be.DeleteUser(ctx, "owner")
	is.True(errors.Is(err, proto.ErrUserOwnsRepos))
	_, err = be.User(ctx, "owner")
	is.NoErr(err)
	_, err = be.Repository(ctx, "owned")
	is.NoErr(err)

	is.NoErr(be.DeleteRepository(ctx, "owned"))
	is.NoErr(be.DeleteUser(ctx, "owner"))
	_, err = be.User(ctx, "owner")
	is.True(errors.Is(err, proto.ErrUserNotFound))

	is.True(errors.Is(be.DeleteUser(ctx, "owner"), proto.ErrUserNotFound))
}

// TestDeleteRepositoryWithoutDirectory verifies that a repository whose
// directory is gone can still be deleted, and that deleting a repository
// that does not exist at all reports it as not found.
func TestDeleteRepositoryWithoutDirectory(t *testing.T) {
	is := is.New(t)
	be, cfg := newTestBackend(t)

	ctx := context.Background()
	ctx = config.WithContext(ctx, cfg)
	ctx = db.WithContext(ctx, be.db)
	ctx = store.WithContext(ctx, be.store)

	_, err := be.CreateRepository(ctx, "orphan", nil, proto.RepositoryOptions{})
	is.NoErr(err)
	is.NoErr(os.RemoveAll(be.repoPath("orphan")))

	is.NoErr(be.DeleteRepository(ctx, "orphan"))
	is.True(errors.Is(be.DeleteRepository(ctx, "orphan"), proto.ErrRepoNotFound))
}
