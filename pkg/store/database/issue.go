package database

import (
	"context"
	"fmt"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/store"
)

type issueStore struct{}

var _ store.IssueStore = (*issueStore)(nil)

// maxIssuesPerRepo caps the number of issues returned to prevent unbounded memory use.
const maxIssuesPerRepo = 10000

// validStatus returns an error if the given status string is not a recognised value.
func validStatus(status string) error {
	switch status {
	case "", "all", "open", "closed":
		return nil
	}
	return fmt.Errorf("invalid status %q: must be open, closed, or all", status)
}

// GetIssueByID implements store.IssueStore.
func (*issueStore) GetIssueByID(ctx context.Context, h db.Handler, id int64) (models.Issue, error) {
	var issue models.Issue
	query := h.Rebind("SELECT * FROM issues WHERE id = ?;")
	err := h.GetContext(ctx, &issue, query, id)
	return issue, db.WrapError(err)
}

// GetIssuesByRepoID implements store.IssueStore.
func (*issueStore) GetIssuesByRepoID(ctx context.Context, h db.Handler, repoID int64, status string) ([]models.Issue, error) {
	if err := validStatus(status); err != nil {
		return nil, err
	}

	var issues []models.Issue
	var query string
	var args []interface{}

	if status == "" || status == "all" {
		query = h.Rebind("SELECT * FROM issues WHERE repo_id = ? ORDER BY created_at DESC LIMIT ?;")
		args = []interface{}{repoID, maxIssuesPerRepo}
	} else {
		query = h.Rebind("SELECT * FROM issues WHERE repo_id = ? AND status = ? ORDER BY created_at DESC LIMIT ?;")
		args = []interface{}{repoID, status, maxIssuesPerRepo}
	}

	err := h.SelectContext(ctx, &issues, query, args...)
	return issues, db.WrapError(err)
}

// CreateIssue implements store.IssueStore.
func (*issueStore) CreateIssue(ctx context.Context, h db.Handler, repoID, userID int64, title, body string) (int64, error) {
	var id int64
	query := h.Rebind(`INSERT INTO issues (repo_id, user_id, title, body, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'open', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP) RETURNING id;`)
	err := h.QueryRowxContext(ctx, query, repoID, userID, title, body).Scan(&id)
	return id, db.WrapError(err)
}

// UpdateIssue implements store.IssueStore.
// A nil body means "do not change the existing body value".
func (*issueStore) UpdateIssue(ctx context.Context, h db.Handler, id, repoID int64, title string, body *string) error {
	var err error
	if body == nil {
		query := h.Rebind(`UPDATE issues SET title = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND repo_id = ?;`)
		_, err = h.ExecContext(ctx, query, title, id, repoID)
	} else {
		query := h.Rebind(`UPDATE issues SET title = ?, body = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND repo_id = ?;`)
		_, err = h.ExecContext(ctx, query, title, *body, id, repoID)
	}
	return db.WrapError(err)
}

// CloseIssue implements store.IssueStore.
func (*issueStore) CloseIssue(ctx context.Context, h db.Handler, id, repoID, closedBy int64) error {
	query := h.Rebind(`UPDATE issues SET status = 'closed', closed_at = CURRENT_TIMESTAMP, closed_by = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND repo_id = ?;`)
	_, err := h.ExecContext(ctx, query, closedBy, id, repoID)
	return db.WrapError(err)
}

// ReopenIssue implements store.IssueStore.
func (*issueStore) ReopenIssue(ctx context.Context, h db.Handler, id, repoID int64) error {
	query := h.Rebind(`UPDATE issues SET status = 'open', closed_at = NULL, closed_by = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND repo_id = ?;`)
	_, err := h.ExecContext(ctx, query, id, repoID)
	return db.WrapError(err)
}

// DeleteIssue implements store.IssueStore.
func (*issueStore) DeleteIssue(ctx context.Context, h db.Handler, id, repoID int64) error {
	query := h.Rebind("DELETE FROM issues WHERE id = ? AND repo_id = ?;")
	_, err := h.ExecContext(ctx, query, id, repoID)
	return db.WrapError(err)
}

// CountIssues implements store.IssueStore.
func (*issueStore) CountIssues(ctx context.Context, h db.Handler, repoID int64, status string) (int64, error) {
	if err := validStatus(status); err != nil {
		return 0, err
	}

	var count int64
	var query string
	var args []interface{}

	if status == "" || status == "all" {
		query = h.Rebind("SELECT COUNT(*) FROM issues WHERE repo_id = ?;")
		args = []interface{}{repoID}
	} else {
		query = h.Rebind("SELECT COUNT(*) FROM issues WHERE repo_id = ? AND status = ?;")
		args = []interface{}{repoID, status}
	}

	err := h.GetContext(ctx, &count, query, args...)
	return count, db.WrapError(err)
}
