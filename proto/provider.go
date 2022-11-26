package proto

// Provider is a repository provider.
type Provider interface {
	// Open opens a repository.
	Open(name string) (RepositoryService, error)
}
