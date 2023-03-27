package backend

import (
	"github.com/charmbracelet/soft-serve/git"
	"golang.org/x/crypto/ssh"
)

// RepositoryStore is an interface for managing repositories.
type RepositoryStore interface {
	// Repository finds the given repository.
	Repository(repo string) (Repository, error)
	// Repositories returns a list of all repositories.
	Repositories() ([]Repository, error)
	// CreateRepository creates a new repository.
	CreateRepository(name string, private bool) (Repository, error)
	// DeleteRepository deletes a repository.
	DeleteRepository(name string) error
	// RenameRepository renames a repository.
	RenameRepository(oldName, newName string) error
}

// RepositoryMetadata is an interface for managing repository metadata.
type RepositoryMetadata interface {
	// Description returns the repository's description.
	Description(repo string) string
	// SetDescription sets the repository's description.
	SetDescription(repo, desc string) error
	// IsPrivate returns whether the repository is private.
	IsPrivate(repo string) bool
	// SetPrivate sets whether the repository is private.
	SetPrivate(repo string, private bool) error
}

// RepositoryAccess is an interface for managing repository access.
type RepositoryAccess interface {
	// AccessLevel returns the access level for the given repository and key.
	AccessLevel(repo string, pk ssh.PublicKey) AccessLevel
	// IsCollaborator returns true if the authorized key is a collaborator on the repository.
	IsCollaborator(pk ssh.PublicKey, repo string) bool
	// AddCollaborator adds the authorized key as a collaborator on the repository.
	AddCollaborator(pk ssh.PublicKey, memo string, repo string) error
	// RemoveCollaborator removes the authorized key as a collaborator on the repository.
	RemoveCollaborator(pk ssh.PublicKey, repo string) error
	// Collaborators returns a list of all collaborators on the repository.
	Collaborators(repo string) ([]string, error)
	// IsAdmin returns true if the authorized key is an admin.
	IsAdmin(pk ssh.PublicKey) bool
	// AddAdmin adds the authorized key as an admin.
	AddAdmin(pk ssh.PublicKey, memo string) error
	// RemoveAdmin removes the authorized key as an admin.
	RemoveAdmin(pk ssh.PublicKey) error
	// Admins returns a list of all admins.
	Admins() ([]string, error)
}

// Repository is a Git repository interface.
type Repository interface {
	// Name returns the repository's name.
	Name() string
	// Description returns the repository's description.
	Description() string
	// IsPrivate returns whether the repository is private.
	IsPrivate() bool
	// Open returns the underlying git.Repository.
	Open() (*git.Repository, error)
}
