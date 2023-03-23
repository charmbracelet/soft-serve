package backend

import "github.com/charmbracelet/soft-serve/git"

// Repository is a Git repository interface.
type Repository interface {
	// Name returns the repository's name.
	Name() string
	// Description returns the repository's description.
	Description() string
	// IsPrivate returns whether the repository is private.
	IsPrivate() bool
	// Repository returns the underlying git.Repository.
	Repository() (*git.Repository, error)
}
