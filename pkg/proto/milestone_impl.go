package proto

import (
	"time"

	"github.com/charmbracelet/soft-serve/pkg/db/models"
)

// milestone is an implementation of the Milestone interface.
type milestone struct {
	model models.Milestone
}

// NewMilestone creates a new Milestone from a models.Milestone.
func NewMilestone(m models.Milestone) Milestone {
	return &milestone{model: m}
}

// ID returns the milestone's ID.
func (m *milestone) ID() int64 { return m.model.ID }

// RepoID returns the milestone's repo ID.
func (m *milestone) RepoID() int64 { return m.model.RepoID }

// Title returns the milestone's title.
func (m *milestone) Title() string { return m.model.Title }

// Description returns the milestone's description.
func (m *milestone) Description() string { return m.model.Description }

// DueDate returns the milestone's due date; zero value if not set.
func (m *milestone) DueDate() time.Time {
	if m.model.DueDate.Valid {
		return m.model.DueDate.Time
	}
	return time.Time{}
}

// IsOpen returns true if the milestone is open (not closed).
func (m *milestone) IsOpen() bool { return !m.model.ClosedAt.Valid }

// IsClosed returns true if the milestone is closed.
func (m *milestone) IsClosed() bool { return m.model.ClosedAt.Valid }

// ClosedAt returns the time the milestone was closed; zero value if open.
func (m *milestone) ClosedAt() time.Time {
	if m.model.ClosedAt.Valid {
		return m.model.ClosedAt.Time
	}
	return time.Time{}
}

// CreatedAt returns the time the milestone was created.
func (m *milestone) CreatedAt() time.Time { return m.model.CreatedAt }

// UpdatedAt returns the time the milestone was last updated.
func (m *milestone) UpdatedAt() time.Time { return m.model.UpdatedAt }
