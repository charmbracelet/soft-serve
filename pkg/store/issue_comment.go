package store

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
)

// IssueCommentStore is an interface for managing issue comments.
type IssueCommentStore interface {
	// GetIssueCommentByID retrieves a comment by its ID.
	GetIssueCommentByID(ctx context.Context, h db.Handler, id int64) (models.IssueComment, error)
	// GetCommentsByIssueID retrieves all comments for an issue, ordered oldest first.
	GetCommentsByIssueID(ctx context.Context, h db.Handler, issueID int64) ([]models.IssueComment, error)
	// CreateIssueComment creates a new comment on an issue.
	CreateIssueComment(ctx context.Context, h db.Handler, issueID, userID int64, body string) (int64, error)
	// UpdateIssueComment updates the body of a comment.
	UpdateIssueComment(ctx context.Context, h db.Handler, id int64, body string) error
	// DeleteIssueComment deletes a comment by its ID.
	DeleteIssueComment(ctx context.Context, h db.Handler, id int64) error
}
