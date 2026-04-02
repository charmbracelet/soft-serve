package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/store"
)

type issueStore struct{}

var _ store.IssueStore = (*issueStore)(nil)

// validStatus returns an error if the given status string is not a recognised value.
func validStatus(status string) error {
	switch status {
	case "", "all", "open", "closed":
		return nil
	}
	return fmt.Errorf("invalid status %q: must be open, closed, or all", status)
}

// buildIssueWhere constructs the JOIN clause, WHERE conditions (without the "WHERE" keyword),
// and argument list for a query scoped to repoID + filter.
// The caller is responsible for appending ORDER BY / LIMIT / OFFSET as needed.
func buildIssueWhere(repoID int64, filter store.IssueFilter) (joins string, conditions []string, args []interface{}) {
	conditions = append(conditions, "issues.repo_id = ?")
	args = append(args, repoID)

	if filter.LabelName != "" {
		joins = "JOIN issue_labels ON issues.id = issue_labels.issue_id " +
			"JOIN labels ON issue_labels.label_id = labels.id"
		conditions = append(conditions, "labels.name = ?")
		args = append(args, filter.LabelName)
	}

	if filter.Status == "open" || filter.Status == "closed" {
		conditions = append(conditions, "issues.status = ?")
		args = append(args, filter.Status)
	}

	if filter.Search != "" {
		conditions = append(conditions, "(issues.title LIKE ? OR issues.body LIKE ?)")
		pattern := "%" + filter.Search + "%"
		args = append(args, pattern, pattern)
	}

	return joins, conditions, args
}

// GetIssueByID implements store.IssueStore.
func (*issueStore) GetIssueByID(ctx context.Context, h db.Handler, id int64) (models.Issue, error) {
	var issue models.Issue
	query := h.Rebind("SELECT * FROM issues WHERE id = ?;")
	err := h.GetContext(ctx, &issue, query, id)
	return issue, db.WrapError(err)
}

// GetIssuesByRepoID implements store.IssueStore.
func (*issueStore) GetIssuesByRepoID(ctx context.Context, h db.Handler, repoID int64, filter store.IssueFilter) ([]models.Issue, error) {
	if err := validStatus(filter.Status); err != nil {
		return nil, err
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = store.DefaultIssueLimit
	}
	page := filter.Page
	if page <= 1 {
		page = 1
	}
	offset := (page - 1) * limit

	joins, conditions, args := buildIssueWhere(repoID, filter)
	where := strings.Join(conditions, " AND ")

	var sb strings.Builder
	sb.WriteString("SELECT issues.* FROM issues ")
	if joins != "" {
		sb.WriteString(joins)
		sb.WriteString(" ")
	}
	sb.WriteString("WHERE ")
	sb.WriteString(where)
	sb.WriteString(" ORDER BY issues.created_at DESC LIMIT ? OFFSET ?;")
	args = append(args, limit, offset)

	var issues []models.Issue
	err := h.SelectContext(ctx, &issues, h.Rebind(sb.String()), args...)
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
func (*issueStore) CountIssues(ctx context.Context, h db.Handler, repoID int64, filter store.IssueFilter) (int64, error) {
	if err := validStatus(filter.Status); err != nil {
		return 0, err
	}

	joins, conditions, args := buildIssueWhere(repoID, filter)
	where := strings.Join(conditions, " AND ")

	var sb strings.Builder
	sb.WriteString("SELECT COUNT(*) FROM issues ")
	if joins != "" {
		sb.WriteString(joins)
		sb.WriteString(" ")
	}
	sb.WriteString("WHERE ")
	sb.WriteString(where)
	sb.WriteString(";")

	var count int64
	err := h.GetContext(ctx, &count, h.Rebind(sb.String()), args...)
	return count, db.WrapError(err)
}
