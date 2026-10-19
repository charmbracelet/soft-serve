package cmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/matryer/is"
)

// repoScopedCommands lists every repo subcommand that names an existing
// repository, with a placeholder for the repository name.
var repoScopedCommands = [][]string{
	{"info", "REPO"},
	{"tree", "REPO"},
	{"blob", "REPO", "README.md"},
	{"commit", "REPO", "HEAD"},
	{"branch", "list", "REPO"},
	{"branch", "default", "REPO"},
	{"branch", "default", "REPO", "main"},
	{"branch", "delete", "REPO", "main"},
	{"tag", "list", "REPO"},
	{"tag", "delete", "REPO", "v1"},
	{"delete", "REPO"},
	{"rename", "REPO", "other"},
	{"collab", "list", "REPO"},
	{"collab", "add", "REPO", "someone"},
	{"collab", "add", "REPO", "someone", "admin-access"},
	{"collab", "remove", "REPO", "someone"},
	{"description", "REPO"},
	{"description", "REPO", "new"},
	{"project-name", "REPO"},
	{"project-name", "REPO", "new"},
	{"private", "REPO"},
	{"private", "REPO", "true"},
	{"hidden", "REPO"},
	{"hidden", "REPO", "true"},
	{"mirror", "REPO"},
	{"mirror", "REPO", "true"},
	{"webhook", "list", "REPO"},
	{"webhook", "create", "REPO", "https://example.com", "-e", "push"},
	{"webhook", "delete", "REPO", "1"},
	{"webhook", "update", "REPO", "1"},
	{"webhook", "deliveries", "list", "REPO", "1"},
	{"webhook", "deliveries", "redeliver", "REPO", "1", "1"},
	{"webhook", "deliveries", "get", "REPO", "1", "1"},
}

func withRepoName(args []string, name string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		if a == "REPO" {
			a = name
		}
		out[i] = a
	}
	return out
}

// assertSameAnswer runs every repo-scoped command against a private
// repository the caller cannot read and against a missing one, and checks
// the caller gets want for both. Any difference would reveal which names
// exist.
func assertSameAnswer(t *testing.T, ctx context.Context, want error) {
	t.Helper()
	for _, args := range repoScopedCommands {
		name := strings.Join(args, " ")
		t.Run(name, func(t *testing.T) {
			is := is.New(t)
			privOut, privErrOut, privErr := runRepoOutput(t, ctx, withRepoName(args, "secret")...)
			missOut, missErrOut, missErr := runRepoOutput(t, ctx, withRepoName(args, "ghost")...)
			if !errors.Is(privErr, want) {
				t.Errorf("private: expected %v, got %v", want, privErr)
			}
			if !errors.Is(missErr, want) {
				t.Errorf("missing: expected %v, got %v", want, missErr)
			}
			is.Equal(privOut, "")
			is.Equal(missOut, privOut)
			is.Equal(missErrOut, privErrOut)
		})
	}
}

func setupSecretRepo(t *testing.T) (context.Context, context.Context, *backend.Backend) {
	t.Helper()
	is := is.New(t)
	ctx, be := newAuthTestContext(t)
	ownerCtx := withUser(t, ctx, be, "owner", false)
	_, err := be.CreateRepository(ownerCtx, "secret", proto.UserFromContext(ownerCtx), proto.RepositoryOptions{Private: true})
	is.NoErr(err)
	_ = withUser(t, ctx, be, "someone", false)
	return ctx, ownerCtx, be
}

// TestAnonymousCannotTellPrivateFromMissing covers callers with no account:
// both a private repository and a missing one answer ErrUnauthorized.
func TestAnonymousCannotTellPrivateFromMissing(t *testing.T) {
	ctx, _, _ := setupSecretRepo(t)
	assertSameAnswer(t, ctx, proto.ErrUnauthorized)
}

// TestRegisteredUserCannotTellPrivateFromMissing covers signed-in users
// without access: both answer ErrRepoNotFound, even though a missing name
// would otherwise grant them read-write access so a push can create it.
func TestRegisteredUserCannotTellPrivateFromMissing(t *testing.T) {
	ctx, _, be := setupSecretRepo(t)
	userCtx := withUser(t, ctx, be, "stranger", false)
	assertSameAnswer(t, userCtx, proto.ErrRepoNotFound)
}

// TestNoAccessCollaboratorCannotTellPrivateFromMissing covers a user who was
// explicitly granted no-access on the private repository.
func TestNoAccessCollaboratorCannotTellPrivateFromMissing(t *testing.T) {
	is := is.New(t)
	ctx, ownerCtx, be := setupSecretRepo(t)
	userCtx := withUser(t, ctx, be, "blocked", false)
	is.NoErr(be.AddCollaborator(ownerCtx, "secret", "blocked", 0))
	assertSameAnswer(t, userCtx, proto.ErrRepoNotFound)
}

// TestReaderGetsUnauthorizedForWrites verifies that a caller who can read a
// repository but not write to it is still told "unauthorized" for writes, as
// existence is not a secret from someone who can read it.
func TestReaderGetsUnauthorizedForWrites(t *testing.T) {
	ctx, be := newAuthTestContext(t)
	ownerCtx := withUser(t, ctx, be, "owner", false)
	_, err := be.CreateRepository(ownerCtx, "public", proto.UserFromContext(ownerCtx), proto.RepositoryOptions{})
	if err != nil {
		t.Fatal(err)
	}

	userCtx := withUser(t, ctx, be, "reader", false)
	for _, args := range [][]string{
		{"delete", "public"},
		{"rename", "public", "other"},
		{"collab", "list", "public"},
		{"description", "public", "new"},
		{"private", "public", "true"},
		{"hidden", "public", "true"},
		{"webhook", "list", "public"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			err := runRepo(t, userCtx, args...)
			if !errors.Is(err, proto.ErrUnauthorized) {
				t.Errorf("expected ErrUnauthorized, got %v", err)
			}
		})
	}

	// Reads still work.
	if err := runRepo(t, userCtx, "description", "public"); err != nil {
		t.Errorf("reader cannot read description: %v", err)
	}
}

// TestAdminCanDeleteOrphanedRepository verifies the read gate lets a server
// admin through to a repository whose directory is gone, so the leftover
// database row can still be deleted.
func TestAdminCanDeleteOrphanedRepository(t *testing.T) {
	is := is.New(t)
	ctx, be := newAuthTestContext(t)
	adminCtx := withUser(t, ctx, be, "root", true)
	_, err := be.CreateRepository(adminCtx, "orphan", proto.UserFromContext(adminCtx), proto.RepositoryOptions{})
	is.NoErr(err)

	cfg := config.FromContext(ctx)
	is.NoErr(os.RemoveAll(filepath.Join(cfg.DataPath, "repos", "orphan.git")))

	// Reads still say not found, even to the admin.
	err = runRepo(t, adminCtx, "info", "orphan")
	is.True(errors.Is(err, proto.ErrRepoNotFound))

	is.NoErr(runRepo(t, adminCtx, "delete", "orphan"))
	repos, err := be.Repositories(ctx)
	is.NoErr(err)
	is.Equal(len(repos), 0)

	// A second delete reports the name as missing.
	err = runRepo(t, adminCtx, "delete", "orphan")
	is.True(errors.Is(err, proto.ErrRepoNotFound))
}

// TestRenameDestinationIsGatedLikeCreate verifies a writer cannot rename
// onto a name they could not create, and gets the same answer create gives.
func TestRenameDestinationIsGatedLikeCreate(t *testing.T) {
	is := is.New(t)
	ctx, be := newAuthTestContext(t)
	ownerCtx := withUser(t, ctx, be, "owner", false)
	otherCtx := withUser(t, ctx, be, "other", false)
	_, err := be.CreateRepository(ownerCtx, "mine", proto.UserFromContext(ownerCtx), proto.RepositoryOptions{})
	is.NoErr(err)
	_, err = be.CreateRepository(otherCtx, "theirs", proto.UserFromContext(otherCtx), proto.RepositoryOptions{})
	is.NoErr(err)

	renameErr := runRepo(t, ownerCtx, "rename", "mine", "theirs")
	createErr := runRepo(t, ownerCtx, "create", "theirs")
	is.True(errors.Is(renameErr, proto.ErrUnauthorized))
	is.Equal(renameErr, createErr)

	// The gate for the source does not leak into the destination check.
	is.NoErr(runRepo(t, ownerCtx, "rename", "mine", "renamed"))
	_, err = be.Repository(ctx, "renamed")
	is.NoErr(err)
}
