package database

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/store"
)

type issueStore struct{}

var _ store.IssueStore = (*issueStore)(nil)

// GetIssueByID implements store.IssueStore.
func (*issueStore) GetIssueByID(ctx context.Context, h db.Handler, id int64) (models.Issue, error) {
	var issue models.Issue
	query := h.Rebind("SELECT * FROM issues WHERE id = ?;")
	err := h.GetContext(ctx, &issue, query, id)
	return issue, db.WrapError(err)
}

// GetIssuesByRepoID implements store.IssueStore.
func (*issueStore) GetIssuesByRepoID(ctx context.Context, h db.Handler, repoID int64, status string) ([]models.Issue, error) {
	var issues []models.Issue
	var query string
	var args []interface{}

	if status == "" || status == "all" {
		query = h.Rebind("SELECT * FROM issues WHERE repo_id = ? ORDER BY created_at DESC;")
		args = []interface{}{repoID}
	} else {
		query = h.Rebind("SELECT * FROM issues WHERE repo_id = ? AND status = ? ORDER BY created_at DESC;")
		args = []interface{}{repoID, status}
	}

	err := h.SelectContext(ctx, &issues, query, args...)
	return issues, db.WrapError(err)
}

// CreateIssue implements store.IssueStore.
func (*issueStore) CreateIssue(ctx context.Context, h db.Handler, repoID, userID int64, title, body string) (int64, error) {
	query := h.Rebind(`INSERT INTO issues (repo_id, user_id, title, body, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'open', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);`)

	result, err := h.ExecContext(ctx, query, repoID, userID, title, body)
	if err != nil {
		return 0, db.WrapError(err)
	}

	return result.LastInsertId()
}

// UpdateIssue implements store.IssueStore.
func (*issueStore) UpdateIssue(ctx context.Context, h db.Handler, id int64, title, body string) error {
	query := h.Rebind(`UPDATE issues SET title = ?, body = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;`)
	_, err := h.ExecContext(ctx, query, title, body, id)
	return db.WrapError(err)
}

// CloseIssue implements store.IssueStore.
func (*issueStore) CloseIssue(ctx context.Context, h db.Handler, id, closedBy int64) error {
	query := h.Rebind(`UPDATE issues SET status = 'closed', closed_at = CURRENT_TIMESTAMP, closed_by = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;`)
	_, err := h.ExecContext(ctx, query, closedBy, id)
	return db.WrapError(err)
}

// ReopenIssue implements store.IssueStore.
func (*issueStore) ReopenIssue(ctx context.Context, h db.Handler, id int64) error {
	query := h.Rebind(`UPDATE issues SET status = 'open', closed_at = NULL, closed_by = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = ?;`)
	_, err := h.ExecContext(ctx, query, id)
	return db.WrapError(err)
}

// DeleteIssue implements store.IssueStore.
func (*issueStore) DeleteIssue(ctx context.Context, h db.Handler, id int64) error {
	query := h.Rebind("DELETE FROM issues WHERE id = ?;")
	_, err := h.ExecContext(ctx, query, id)
	return db.WrapError(err)
}

// CountIssues implements store.IssueStore.
func (*issueStore) CountIssues(ctx context.Context, h db.Handler, repoID int64, status string) (int64, error) {
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
