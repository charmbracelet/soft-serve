package proto

import "github.com/charmbracelet/soft-serve/pkg/access"

// CollaboratorType is a collaborator type.
type CollaboratorType int

const (
	// CollaboratorTypeUser is a user collaborator.
	CollaboratorTypeUser CollaboratorType = iota
	// CollaboratorTypeTeam is a team collaborator.
	CollaboratorTypeTeam
)

// Collaborator is an interface representing a collaborator.
type Collaborator interface {
	// ID returns the collaborator's ID.
	ID() int64
	// Type returns the collaborator's type.
	Type() CollaboratorType
	// RepoID returns the repository ID.
	RepoID() int64
	// AccessLevel returns the collaborator's access level for the repository.
	AccessLevel() access.AccessLevel
}
