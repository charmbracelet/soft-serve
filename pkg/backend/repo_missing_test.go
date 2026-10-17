package backend

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/matryer/is"
)

// TestMissingRepositoryAccessors verifies every per-repository getter and
// setter reports a missing repository as proto.ErrRepoNotFound rather than
// leaking a database or filesystem error, or silently succeeding.
func TestMissingRepositoryAccessors(t *testing.T) {
	be, cfg := newTestBackend(t)
	ctx := context.Background()

	tests := map[string]func() error{
		"Description": func() error {
			_, err := be.Description(ctx, "ghost")
			return err
		},
		"IsMirror": func() error {
			_, err := be.IsMirror(ctx, "ghost")
			return err
		},
		"IsPrivate": func() error {
			_, err := be.IsPrivate(ctx, "ghost")
			return err
		},
		"IsHidden": func() error {
			_, err := be.IsHidden(ctx, "ghost")
			return err
		},
		"ProjectName": func() error {
			_, err := be.ProjectName(ctx, "ghost")
			return err
		},
		"SetHidden":      func() error { return be.SetHidden(ctx, "ghost", true) },
		"SetMirror":      func() error { return be.SetMirror(ctx, "ghost", true) },
		"SetDescription": func() error { return be.SetDescription(ctx, "ghost", "desc") },
		"SetPrivate":     func() error { return be.SetPrivate(ctx, "ghost", true) },
		"SetProjectName": func() error { return be.SetProjectName(ctx, "ghost", "name") },
		"Collaborators": func() error {
			_, err := be.Collaborators(ctx, "ghost")
			return err
		},
		"AddCollaborator":    func() error { return be.AddCollaborator(ctx, "ghost", "admin", access.ReadOnlyAccess) },
		"RemoveCollaborator": func() error { return be.RemoveCollaborator(ctx, "ghost", "admin") },
	}

	for name, fn := range tests {
		t.Run(name, func(t *testing.T) {
			err := fn()
			if !errors.Is(err, proto.ErrRepoNotFound) {
				t.Fatalf("expected ErrRepoNotFound, got %v", err)
			}
			if strings.Contains(err.Error(), cfg.DataPath) {
				t.Fatalf("error leaks the server data path: %v", err)
			}
		})
	}

	// Nothing should have been created along the way.
	if _, err := os.Stat(be.repoPath("ghost")); err == nil {
		t.Fatal("a missing repository was created by a setter")
	}
}

// TestSetDescriptionHidesFilesystemError verifies a filesystem failure while
// updating a repository does not echo the server path back to the caller.
func TestSetDescriptionHidesFilesystemError(t *testing.T) {
	is := is.New(t)
	be, cfg := newTestBackend(t)
	ctx := context.Background()

	_, err := be.CreateRepository(ctx, "repo", nil, proto.RepositoryOptions{})
	is.NoErr(err)

	// Make the description path unwritable by turning it into a directory.
	descPath := be.repoPath("repo") + "/description"
	is.NoErr(os.Remove(descPath))
	is.NoErr(os.Mkdir(descPath, 0o755))

	err = be.SetDescription(ctx, "repo", "desc")
	is.True(err != nil)
	is.True(!strings.Contains(err.Error(), cfg.DataPath))
}
