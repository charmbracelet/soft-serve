package proto

import "time"

// Issue is an issue interface.
type Issue interface {
	// ID returns the issue's ID.
	ID() int64
	// RepoID returns the ID of the repository the issue belongs to.
	RepoID() int64
	// UserID returns the ID of the user who created the issue.
	UserID() int64
	// Title returns the issue's title.
	Title() string
	// Body returns the issue's body.
	Body() string
	// Status returns the issue's status (open or closed).
	Status() string
	// IsOpen returns whether the issue is open.
	IsOpen() bool
	// IsClosed returns whether the issue is closed.
	IsClosed() bool
	// CreatedAt returns the time the issue was created.
	CreatedAt() time.Time
	// UpdatedAt returns the time the issue was last updated.
	UpdatedAt() time.Time
	// ClosedAt returns the time the issue was closed, or zero time if open.
	ClosedAt() time.Time
	// ClosedBy returns the ID of the user who closed the issue, or 0 if open.
	ClosedBy() int64
}

