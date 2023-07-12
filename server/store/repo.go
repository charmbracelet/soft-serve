package store

import (
	"time"

	"github.com/charmbracelet/soft-serve/git"
)

// Repository is a Git repository interface.
type Repository interface {
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
}
