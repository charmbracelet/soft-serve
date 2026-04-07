package proto

import "time"

// Milestone is a repository milestone interface.
type Milestone interface {
	// ID returns the milestone's ID.
	ID() int64
	// RepoID returns the ID of the repository the milestone belongs to.
	RepoID() int64
	// Title returns the milestone's title.
	Title() string
	// Description returns the milestone's description.
	Description() string
	// DueDate returns the milestone's due date; zero value if not set.
	DueDate() time.Time
	// IsOpen returns true if the milestone is open.
	IsOpen() bool
	// IsClosed returns true if the milestone is closed.
	IsClosed() bool
	// ClosedAt returns the time the milestone was closed; zero value if open.
	ClosedAt() time.Time
	// CreatedAt returns the time the milestone was created.
	CreatedAt() time.Time
	// UpdatedAt returns the time the milestone was last updated.
	UpdatedAt() time.Time
}
