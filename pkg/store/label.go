package store

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
)

// LabelStore is an interface for managing labels and issue-label associations.
type LabelStore interface {
	// GetLabelByID retrieves a label by its ID.
	GetLabelByID(ctx context.Context, h db.Handler, id int64) (models.Label, error)
	// GetLabelByName retrieves a label by repo ID and name.
	GetLabelByName(ctx context.Context, h db.Handler, repoID int64, name string) (models.Label, error)
	// GetLabelsByRepoID retrieves all labels for a repository.
	GetLabelsByRepoID(ctx context.Context, h db.Handler, repoID int64) ([]models.Label, error)
	// GetLabelsByIssueID retrieves all labels attached to an issue.
	GetLabelsByIssueID(ctx context.Context, h db.Handler, issueID int64) ([]models.Label, error)
	// CreateLabel creates a new label for a repository.
	CreateLabel(ctx context.Context, h db.Handler, repoID int64, name, color, description string) (int64, error)
	// UpdateLabel updates a label's name, color, and description.
	UpdateLabel(ctx context.Context, h db.Handler, id, repoID int64, name, color, description string) error
	// DeleteLabel deletes a label by its ID.
	DeleteLabel(ctx context.Context, h db.Handler, id, repoID int64) error
	// AddLabelToIssue attaches a label to an issue.
	AddLabelToIssue(ctx context.Context, h db.Handler, issueID, labelID int64) error
	// RemoveLabelFromIssue detaches a label from an issue.
	RemoveLabelFromIssue(ctx context.Context, h db.Handler, issueID, labelID int64) error
}
