package proto

import (
	"time"

	"github.com/charmbracelet/soft-serve/git"
)

// Repository is a Git repository interface.
type Repository interface {
	// ID returns the repository's ID.
	ID() int64
	// Name returns the repository's name.
	Name() string
	// ProjectName returns the repository's project name.
	ProjectName() string
	// Description returns the repository's description.
	Description() string
	// IsPrivate returns whether the repository is private.
	IsPrivate() bool
	// IsMirror returns whether the repository is a mirror.
	IsMirror() bool
	// IsHidden returns whether the repository is hidden.
	IsHidden() bool
	// UserID returns the ID of the user who owns the repository.
	// It returns 0 if the repository is not owned by a user.
	UserID() int64
	// CreatedAt returns the time the repository was created.
	CreatedAt() time.Time
	// UpdatedAt returns the time the repository was last updated.
	// If the repository has never been updated, it returns the time it was created.
	UpdatedAt() time.Time
	// Open returns the underlying git.Repository.
	Open() (*git.Repository, error)
}

// RepositoryOptions are options for creating a new repository.
type RepositoryOptions struct {
	Private     bool
	Description string
	ProjectName string
	Mirror      bool
	Hidden      bool
	LFS         bool
	LFSEndpoint string
}

// RepositoryDefaultBranch returns the default branch of a repository.
func RepositoryDefaultBranch(repo Repository) (string, error) {
	r, err := repo.Open()
	if err != nil {
		return "", err //nolint:wrapcheck
	}

	ref, err := r.HEAD()
	if err != nil {
		return "", err //nolint:wrapcheck
	}

	return ref.Name().Short(), nil
}
