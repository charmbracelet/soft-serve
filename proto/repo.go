package proto

// Repository is Git repository.
type Repository interface {
	Name() string
	ProjectName() string
	Description() string
	IsPrivate() bool
}

// RepositoryService is a service for managing repositories metadata.
type RepositoryService interface {
	Repository
	SetProjectName(string) error
	SetDescription(string) error
	SetPrivate(bool) error
}
