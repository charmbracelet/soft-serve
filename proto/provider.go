package proto

// Provider is a Git repository provider.
type Provider interface {
	// Open opens a repository.
	Open(name string) (Repository, error)
	// ListRepos lists all repositories.
	ListRepos() ([]Metadata, error)
	// Create creates a new repository.
	Create(name string, projectName string, description string, isPrivate bool) error
	// Delete deletes a repository.
	Delete(name string) error
	// Rename renames a repository.
	Rename(name string, newName string) error
	// SetProjectName sets a repository's project name.
	SetProjectName(name string, projectName string) error
	// SetDescription sets a repository's description.
	SetDescription(name string, description string) error
	// SetPrivate sets a repository's private flag.
	SetPrivate(name string, isPrivate bool) error
	// SetDefaultBranch sets a repository's default branch.
	SetDefaultBranch(name string, branch string) error
}

// MetadataProvider is a Git repository metadata provider.
type MetadataProvider interface {
	// Metadata gets a repository's metadata.
	Metadata(name string) (Metadata, error)
}
