package database

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/store"
)

type assigneeStore struct{}

var _ store.AssigneeStore = (*assigneeStore)(nil)

// GetAssigneesByIssueID implements store.AssigneeStore.
func (*assigneeStore) GetAssigneesByIssueID(ctx context.Context, h db.Handler, issueID int64) ([]models.User, error) {
	var users []models.User
	query := h.Rebind(`SELECT users.* FROM users
		JOIN issue_assignees ON users.id = issue_assignees.user_id
		WHERE issue_assignees.issue_id = ?
		ORDER BY users.username ASC;`)
	err := h.SelectContext(ctx, &users, query, issueID)
	return users, db.WrapError(err)
}

// AddAssignee implements store.AssigneeStore.
func (*assigneeStore) AddAssignee(ctx context.Context, h db.Handler, issueID, userID int64) error {
	query := h.Rebind("INSERT INTO issue_assignees (issue_id, user_id) VALUES (?, ?);")
	_, err := h.ExecContext(ctx, query, issueID, userID)
	return db.WrapError(err)
}

// RemoveAssignee implements store.AssigneeStore.
func (*assigneeStore) RemoveAssignee(ctx context.Context, h db.Handler, issueID, userID int64) error {
	query := h.Rebind("DELETE FROM issue_assignees WHERE issue_id = ? AND user_id = ?;")
	_, err := h.ExecContext(ctx, query, issueID, userID)
	return db.WrapError(err)
}
