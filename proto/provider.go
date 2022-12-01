package proto

// Provider is a Git repository provider.
type Provider interface {
	// Open opens a repository.
	Open(name string) (Repository, error)
	// ListRepos lists all repositories.
	ListRepos() ([]Metadata, error)
}

// MetadataProvider is a Git repository metadata provider.
type MetadataProvider interface {
	// Metadata gets a repository's metadata.
	Metadata(name string) (Metadata, error)
}
