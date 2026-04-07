package proto

import (
	"time"

	"github.com/charmbracelet/soft-serve/pkg/db/models"
)

// label is an implementation of the Label interface.
type label struct {
	model models.Label
}

// NewLabel creates a new Label from a models.Label.
func NewLabel(m models.Label) Label {
	return &label{model: m}
}

// ID returns the label's ID.
func (l *label) ID() int64 { return l.model.ID }

// RepoID returns the label's repo ID.
func (l *label) RepoID() int64 { return l.model.RepoID }

// Name returns the label's name.
func (l *label) Name() string { return l.model.Name }

// Color returns the label's color.
func (l *label) Color() string { return l.model.Color }

// Description returns the label's description.
func (l *label) Description() string { return l.model.Description }

// CreatedAt returns the time the label was created.
func (l *label) CreatedAt() time.Time { return l.model.CreatedAt }
