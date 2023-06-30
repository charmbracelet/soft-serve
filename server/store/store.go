package store

import (
	"context"

	"github.com/go-git/go-billy/v5"
)

// RepositoryOptions are options for creating a new repository.
type RepositoryOptions struct {
	Private     bool
	Description string
	ProjectName string
	Mirror      bool
	Hidden      bool
}

// Store is an interface for managing repositories and repository metadata.
type Store interface {
	// Filesystem returns the underlying git filesystem.
	Filesystem() billy.Filesystem
	// Repository finds the given repository.
	Repository(ctx context.Context, repo string) (Repository, error)
	// CountRepositories returns the number of repositories.
	CountRepositories(ctx context.Context) (uint64, error)
	// Repositories returns a list of all repositories.
	Repositories(ctx context.Context, page int, perPage int) ([]Repository, error)
	// CreateRepository creates a new repository.
	CreateRepository(ctx context.Context, name string, opts RepositoryOptions) (Repository, error)
	// ImportRepository creates a new repository from a Git repository.
	ImportRepository(ctx context.Context, name string, remote string, opts RepositoryOptions) (Repository, error)
	// DeleteRepository deletes a repository.
	DeleteRepository(ctx context.Context, name string) error
	// RenameRepository renames a repository.
	RenameRepository(ctx context.Context, oldName, newName string) error
	// ProjectName returns the repository's project name.
	ProjectName(ctx context.Context, repo string) (string, error)
	// SetProjectName sets the repository's project name.
	SetProjectName(ctx context.Context, repo, name string) error
	// Description returns the repository's description.
	Description(ctx context.Context, repo string) (string, error)
	// SetDescription sets the repository's description.
	SetDescription(ctx context.Context, repo, desc string) error
	// IsPrivate returns whether the repository is private.
	IsPrivate(ctx context.Context, repo string) (bool, error)
	// SetPrivate sets whether the repository is private.
	SetPrivate(ctx context.Context, repo string, private bool) error
	// IsMirror returns whether the repository is a mirror.
	IsMirror(ctx context.Context, repo string) (bool, error)
	// IsHidden returns whether the repository is hidden.
	IsHidden(ctx context.Context, repo string) (bool, error)
	// SetHidden sets whether the repository is hidden.
	SetHidden(ctx context.Context, repo string, hidden bool) error
	// Touch updates the repository's last activity time.
	Touch(ctx context.Context, repo string) error
}
