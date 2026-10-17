package backend

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/matryer/is"
)

// TestOrphanRepoDirectoryIsNotClaimable verifies that a repository directory
// with no database row is treated as taken rather than missing. Otherwise a
// user who can create repositories could claim it, and on a case-insensitive
// filesystem "claim it" means becoming the owner of somebody else's
// repository.
func TestOrphanRepoDirectoryIsNotClaimable(t *testing.T) {
	is := is.New(t)
	be, cfg := newTestBackend(t)
	ctx := context.Background()

	user, err := be.CreateUser(ctx, "alice", proto.UserOptions{})
	is.NoErr(err)

	orphan := filepath.Join(cfg.DataPath, "repos", "orphan.git")
	is.NoErr(os.MkdirAll(orphan, 0o755))

	is.Equal(be.AccessLevelForUser(ctx, "orphan", user), access.NoAccess)
	is.Equal(be.AccessLevelForUser(ctx, "orphan", nil), access.NoAccess)

	_, err = be.CreateRepository(ctx, "orphan", user, proto.RepositoryOptions{})
	is.True(errors.Is(err, proto.ErrRepoExist))

	// A genuinely new name is still creatable.
	is.Equal(be.AccessLevelForUser(ctx, "fresh", user), access.ReadWriteAccess)
}

// TestCaseVariantOfExistingRepo verifies that a name differing only by case
// from an existing private repository cannot be used to create or gain
// access to it. On a case-sensitive filesystem the two names are distinct
// repositories and the variant is simply creatable; on a case-insensitive
// one they share a directory and the variant must be refused.
func TestCaseVariantOfExistingRepo(t *testing.T) {
	is := is.New(t)
	be, cfg := newTestBackend(t)
	ctx := context.Background()

	owner, err := be.CreateUser(ctx, "owner", proto.UserOptions{})
	is.NoErr(err)
	attacker, err := be.CreateUser(ctx, "attacker", proto.UserOptions{})
	is.NoErr(err)

	_, err = be.CreateRepository(ctx, "secret", owner, proto.RepositoryOptions{Private: true})
	is.NoErr(err)

	_, statErr := os.Stat(filepath.Join(cfg.DataPath, "repos", "SECRET.git"))
	caseInsensitive := statErr == nil

	level := be.AccessLevelForUser(ctx, "SECRET", attacker)
	_, createErr := be.CreateRepository(ctx, "SECRET", attacker, proto.RepositoryOptions{})

	if caseInsensitive {
		is.Equal(level, access.NoAccess)
		is.True(errors.Is(createErr, proto.ErrRepoExist))
	} else {
		is.Equal(level, access.ReadWriteAccess)
		is.NoErr(createErr)
	}

	// Whatever the filesystem did, the original is untouched and still private.
	r, err := be.Repository(ctx, "secret")
	is.NoErr(err)
	is.True(r.IsPrivate())
	is.Equal(r.UserID(), owner.ID())
	is.Equal(be.AccessLevelForUser(ctx, "secret", attacker), access.NoAccess)
}

// TestConcurrentCreateSameName verifies that racing creates of one name yield
// exactly one repository; the directory claim is what arbitrates.
func TestConcurrentCreateSameName(t *testing.T) {
	is := is.New(t)
	be, cfg := newTestBackend(t)
	ctx := context.Background()

	const workers = 8
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := be.CreateRepository(ctx, "raced", nil, proto.RepositoryOptions{})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)

	var created, existed int
	for err := range errs {
		switch {
		case err == nil:
			created++
		case errors.Is(err, proto.ErrRepoExist):
			existed++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}
	is.Equal(created, 1)
	is.Equal(existed, workers-1)

	repos, err := be.Repositories(ctx)
	is.NoErr(err)
	is.Equal(len(repos), 1)
	_, err = os.Stat(filepath.Join(cfg.DataPath, "repos", "raced.git", "HEAD"))
	is.NoErr(err)
}

// TestRegisterRepositoryAdoptsClaimedPath covers the import flow: the
// directory is claimed and filled before the database row exists, and
// registering it must not refuse its own directory.
func TestRegisterRepositoryAdoptsClaimedPath(t *testing.T) {
	is := is.New(t)
	be, cfg := newTestBackend(t)
	ctx := context.Background()

	owner, err := be.CreateUser(ctx, "importer", proto.UserOptions{})
	is.NoErr(err)

	rp := filepath.Join(cfg.DataPath, "repos", "imported.git")
	is.NoErr(claimRepoPath(rp))
	is.True(errors.Is(claimRepoPath(rp), proto.ErrRepoExist))
	_, err = git.Init(rp, true)
	is.NoErr(err)

	r, err := be.registerRepository(ctx, "imported", owner, proto.RepositoryOptions{Private: true})
	is.NoErr(err)
	is.Equal(r.UserID(), owner.ID())
	is.True(r.IsPrivate())
	is.Equal(be.AccessLevelForUser(ctx, "imported", owner), access.AdminAccess)

	// The name is now taken on both axes.
	_, err = be.CreateRepository(ctx, "imported", owner, proto.RepositoryOptions{})
	is.True(errors.Is(err, proto.ErrRepoExist))
	_, err = be.Repository(ctx, "imported")
	is.NoErr(err)
}
