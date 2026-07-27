package backend

import (
	"context"
	"testing"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/matryer/is"
)

// TestCreateRepositoryAnonymousOwner verifies that a repository created with
// a nil user (an anon-access/allow-keyless override) is owned by the
// lowest-ID admin rather than failing the NOT NULL repos.user_id constraint.
func TestCreateRepositoryAnonymousOwner(t *testing.T) {
	is := is.New(t)
	be, _ := newTestBackend(t)
	ctx := context.Background()

	admin, err := be.User(ctx, "admin")
	is.NoErr(err)

	repo, err := be.CreateRepository(ctx, "anon-repo", nil, proto.RepositoryOptions{})
	is.NoErr(err)
	is.Equal(repo.UserID(), admin.ID())
}

// TestDefaultAdminUserIDNoAdmin verifies that defaultAdminUserID surfaces an
// error rather than silently returning a zero user ID (which would violate
// the NOT NULL repos.user_id constraint) when the database has no admin.
func TestDefaultAdminUserIDNoAdmin(t *testing.T) {
	is := is.New(t)
	be, _ := newTestBackend(t)
	ctx := context.Background()

	err := be.db.TransactionContext(ctx, func(tx *db.Tx) error {
		if _, err := tx.ExecContext(ctx, "UPDATE users SET admin = false"); err != nil {
			return err
		}

		_, err := be.defaultAdminUserID(ctx, tx)
		return err
	})
	is.True(err != nil)
}
