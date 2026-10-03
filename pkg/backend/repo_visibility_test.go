package backend

import (
	"context"
	"testing"

	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/matryer/is"
)

// TestRepositoryReflectsDatabase verifies that access checks see a
// visibility change as soon as it is committed, whoever committed it
// (GHSA-f3fj-9625-rv3m).
func TestRepositoryReflectsDatabase(t *testing.T) {
	is := is.New(t)
	be, _ := newTestBackend(t)
	ctx := context.Background()

	_, err := be.CreateRepository(ctx, "repo", nil, proto.RepositoryOptions{})
	is.NoErr(err)
	is.Equal(be.AccessLevelForUser(ctx, "repo", nil), access.ReadOnlyAccess)

	is.NoErr(be.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return be.store.SetRepoIsPrivateByName(ctx, tx, "repo", true)
	}))

	r, err := be.Repository(ctx, "repo")
	is.NoErr(err)
	is.True(r.IsPrivate())
	is.Equal(be.AccessLevelForUser(ctx, "repo", nil), access.NoAccess)
}
