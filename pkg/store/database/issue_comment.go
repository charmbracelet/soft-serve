package database

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/store"
)

type issueCommentStore struct{}

var _ store.IssueCommentStore = (*issueCommentStore)(nil)

// GetIssueCommentByID implements store.IssueCommentStore.
func (*issueCommentStore) GetIssueCommentByID(ctx context.Context, h db.Handler, id int64) (models.IssueComment, error) {
	var c models.IssueComment
	query := h.Rebind("SELECT * FROM issue_comments WHERE id = ?;")
	err := h.GetContext(ctx, &c, query, id)
	return c, db.WrapError(err)
}

// GetCommentsByIssueID implements store.IssueCommentStore.
func (*issueCommentStore) GetCommentsByIssueID(ctx context.Context, h db.Handler, issueID int64) ([]models.IssueComment, error) {
	var comments []models.IssueComment
	query := h.Rebind("SELECT * FROM issue_comments WHERE issue_id = ? ORDER BY created_at ASC;")
	err := h.SelectContext(ctx, &comments, query, issueID)
	return comments, db.WrapError(err)
}

// CreateIssueComment implements store.IssueCommentStore.
func (*issueCommentStore) CreateIssueComment(ctx context.Context, h db.Handler, issueID, userID int64, body string) (int64, error) {
	var id int64
	query := h.Rebind(`INSERT INTO issue_comments (issue_id, user_id, body, created_at, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP) RETURNING id;`)
	err := h.QueryRowxContext(ctx, query, issueID, userID, body).Scan(&id)
	return id, db.WrapError(err)
}

// UpdateIssueComment implements store.IssueCommentStore.
func (*issueCommentStore) UpdateIssueComment(ctx context.Context, h db.Handler, id int64, body string) error {
	query := h.Rebind(`UPDATE issue_comments SET body = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;`)
	_, err := h.ExecContext(ctx, query, body, id)
	return db.WrapError(err)
}

// DeleteIssueComment implements store.IssueCommentStore.
func (*issueCommentStore) DeleteIssueComment(ctx context.Context, h db.Handler, id int64) error {
	query := h.Rebind("DELETE FROM issue_comments WHERE id = ?;")
	_, err := h.ExecContext(ctx, query, id)
	return db.WrapError(err)
}
