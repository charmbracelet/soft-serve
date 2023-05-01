package backend

import (
	"time"

	"github.com/charmbracelet/soft-serve/git"
)

// RepositoryOptions are options for creating a new repository.
type RepositoryOptions struct {
	Private     bool
	Description string
	ProjectName string
	Mirror      bool
	Hidden      bool
}

// RepositoryStore is an interface for managing repositories.
type RepositoryStore interface {
	// Repository finds the given repository.
	Repository(repo string) (Repository, error)
	// Repositories returns a list of all repositories.
	Repositories() ([]Repository, error)
	// CreateRepository creates a new repository.
	CreateRepository(name string, opts RepositoryOptions) (Repository, error)
	// ImportRepository creates a new repository from a Git repository.
	ImportRepository(name string, remote string, opts RepositoryOptions) (Repository, error)
	// DeleteRepository deletes a repository.
	DeleteRepository(name string) error
	// RenameRepository renames a repository.
	RenameRepository(oldName, newName string) error
}

// RepositoryMetadata is an interface for managing repository metadata.
type RepositoryMetadata interface {
	// ProjectName returns the repository's project name.
	ProjectName(repo string) (string, error)
	// SetProjectName sets the repository's project name.
	SetProjectName(repo, name string) error
	// Description returns the repository's description.
	Description(repo string) (string, error)
	// SetDescription sets the repository's description.
	SetDescription(repo, desc string) error
	// IsPrivate returns whether the repository is private.
	IsPrivate(repo string) (bool, error)
	// SetPrivate sets whether the repository is private.
	SetPrivate(repo string, private bool) error
	// IsMirror returns whether the repository is a mirror.
	IsMirror(repo string) (bool, error)
	// IsHidden returns whether the repository is hidden.
	IsHidden(repo string) (bool, error)
	// SetHidden sets whether the repository is hidden.
	SetHidden(repo string, hidden bool) error
}

// RepositoryAccess is an interface for managing repository access.
type RepositoryAccess interface {
	IsCollaborator(repo string, username string) (bool, error)
	// AddCollaborator adds the authorized key as a collaborator on the repository.
	AddCollaborator(repo string, username string) error
	// RemoveCollaborator removes the authorized key as a collaborator on the repository.
	RemoveCollaborator(repo string, username string) error
	// Collaborators returns a list of all collaborators on the repository.
	Collaborators(repo string) ([]string, error)
}

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
