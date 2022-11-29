package types

import (
	"time"
)

// Repo is a repository database model.
type Repo struct {
	ID          int
	Name        string
	ProjectName string
	Description string
	Private     bool
	CreatedAt   *time.Time
	UpdatedAt   *time.Time
}

// String returns the name of the repository.
func (r *Repo) String() string {
	return r.Name
}
