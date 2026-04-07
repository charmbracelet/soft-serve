package proto

import "time"

// Label is a repository label interface.
type Label interface {
	// ID returns the label's ID.
	ID() int64
	// RepoID returns the ID of the repository the label belongs to.
	RepoID() int64
	// Name returns the label's name.
	Name() string
	// Color returns the label's hex color string (may be empty).
	Color() string
	// Description returns the label's description (may be empty).
	Description() string
	// CreatedAt returns the time the label was created.
	CreatedAt() time.Time
}
