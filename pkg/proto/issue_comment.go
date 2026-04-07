package proto

import "time"

// IssueComment is the interface for an issue comment.
type IssueComment interface {
	// ID returns the comment's ID.
	ID() int64
	// IssueID returns the ID of the issue this comment belongs to.
	IssueID() int64
	// UserID returns the ID of the user who wrote the comment.
	UserID() int64
	// Body returns the comment body.
	Body() string
	// CreatedAt returns when the comment was created.
	CreatedAt() time.Time
	// UpdatedAt returns when the comment was last updated.
	UpdatedAt() time.Time
}
