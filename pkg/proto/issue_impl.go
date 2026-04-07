package proto

import (
	"time"

	"github.com/charmbracelet/soft-serve/pkg/db/models"
)

// issue is an implementation of the Issue interface.
type issue struct {
	model models.Issue
}

// NewIssue creates a new Issue from a models.Issue.
func NewIssue(m models.Issue) Issue {
	return &issue{model: m}
}

// ID returns the issue's ID.
func (i *issue) ID() int64 {
	return i.model.ID
}

// RepoID returns the ID of the repository the issue belongs to.
func (i *issue) RepoID() int64 {
	return i.model.RepoID
}

// UserID returns the ID of the user who created the issue.
func (i *issue) UserID() int64 {
	return i.model.UserID
}

// Title returns the issue's title.
func (i *issue) Title() string {
	return i.model.Title
}

// Body returns the issue's body.
func (i *issue) Body() string {
	if i.model.Body.Valid {
		return i.model.Body.String
	}
	return ""
}

// Status returns the issue's status (open or closed).
func (i *issue) Status() string {
	return i.model.Status
}

// CreatedAt returns the time the issue was created.
func (i *issue) CreatedAt() time.Time {
	return i.model.CreatedAt
}

// UpdatedAt returns the time the issue was last updated.
func (i *issue) UpdatedAt() time.Time {
	return i.model.UpdatedAt
}

// ClosedAt returns the time the issue was closed, or zero time if open.
func (i *issue) ClosedAt() time.Time {
	if i.model.ClosedAt.Valid {
		return i.model.ClosedAt.Time
	}
	return time.Time{}
}

// ClosedBy returns the ID of the user who closed the issue, or 0 if open.
func (i *issue) ClosedBy() int64 {
	if i.model.ClosedBy.Valid {
		return i.model.ClosedBy.Int64
	}
	return 0
}

// IsOpen returns whether the issue is open.
func (i *issue) IsOpen() bool {
	return i.model.Status == "open"
}

// IsClosed returns whether the issue is closed.
func (i *issue) IsClosed() bool {
	return i.model.Status == "closed"
}
