package backend

import (
	"context"
	"testing"

	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/matryer/is"
)

// TestReadable verifies the single lookup behind every access gate: the
// repository comes back only when the caller may read it, so a private
// repository and a missing one look identical, while the level still says
// whether a missing name may be created.
func TestReadable(t *testing.T) {
	is := is.New(t)
	be, _ := newTestBackend(t)
	ctx := context.Background()

	owner, err := be.CreateUser(ctx, "owner", proto.UserOptions{})
	is.NoErr(err)
	stranger, err := be.CreateUser(ctx, "stranger", proto.UserOptions{})
	is.NoErr(err)
	_, err = be.CreateRepository(ctx, "secret", owner, proto.RepositoryOptions{Private: true})
	is.NoErr(err)
	_, err = be.CreateRepository(ctx, "public", owner, proto.RepositoryOptions{})
	is.NoErr(err)

	// Anonymous: both nil; the level is the anonymous default either way.
	r, level := be.Readable(ctx, "secret", nil)
	is.Equal(r, nil)
	is.Equal(level, access.NoAccess)
	r, level = be.Readable(ctx, "ghost", nil)
	is.Equal(r, nil)
	is.Equal(level, access.ReadOnlyAccess)

	// Registered stranger: both nil; a missing name may be created.
	r, level = be.Readable(ctx, "secret", stranger)
	is.Equal(r, nil)
	is.Equal(level, access.NoAccess)
	r, level = be.Readable(ctx, "ghost", stranger)
	is.Equal(r, nil)
	is.Equal(level, access.ReadWriteAccess)

	// Readers get the repository.
	r, level = be.Readable(ctx, "public", nil)
	is.True(r != nil)
	is.Equal(level, access.ReadOnlyAccess)
	r, level = be.Readable(ctx, "secret", owner)
	is.True(r != nil)
	is.Equal(level, access.AdminAccess)

	// AccessLevelForUser is the same computation.
	is.Equal(be.AccessLevelForUser(ctx, "secret", stranger), access.NoAccess)
	is.Equal(be.AccessLevelForUser(ctx, "ghost", stranger), access.ReadWriteAccess)
	is.Equal(be.AccessLevelForUser(ctx, "public", nil), access.ReadOnlyAccess)
}

// TestBaseAccessLevelByPublicKey verifies the baseline level does not
// depend on any repository and matches the level for a missing name.
func TestBaseAccessLevelByPublicKey(t *testing.T) {
	is := is.New(t)
	be, _ := newTestBackend(t)
	ctx := context.Background()

	// No key and no account: the anonymous level.
	is.Equal(be.BaseAccessLevelByPublicKey(ctx, nil), access.ReadOnlyAccess)
	is.Equal(be.BaseAccessLevelByPublicKey(ctx, nil), be.AccessLevelByPublicKey(ctx, "ghost", nil))

	is.NoErr(be.SetAnonAccess(ctx, access.NoAccess))
	is.Equal(be.BaseAccessLevelByPublicKey(ctx, nil), access.NoAccess)
}
