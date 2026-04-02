package store

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
)

// AssigneeStore is an interface for managing issue assignees.
type AssigneeStore interface {
	// GetAssigneesByIssueID retrieves all users assigned to an issue.
	GetAssigneesByIssueID(ctx context.Context, h db.Handler, issueID int64) ([]models.User, error)
	// AddAssignee assigns a user to an issue.
	AddAssignee(ctx context.Context, h db.Handler, issueID, userID int64) error
	// RemoveAssignee removes a user from an issue.
	RemoveAssignee(ctx context.Context, h db.Handler, issueID, userID int64) error
}
