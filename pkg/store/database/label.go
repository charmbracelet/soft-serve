package database

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/store"
)

type labelStore struct{}

var _ store.LabelStore = (*labelStore)(nil)

// GetLabelByID implements store.LabelStore.
func (*labelStore) GetLabelByID(ctx context.Context, h db.Handler, id int64) (models.Label, error) {
	var l models.Label
	query := h.Rebind("SELECT * FROM labels WHERE id = ?;")
	err := h.GetContext(ctx, &l, query, id)
	return l, db.WrapError(err)
}

// GetLabelByName implements store.LabelStore.
func (*labelStore) GetLabelByName(ctx context.Context, h db.Handler, repoID int64, name string) (models.Label, error) {
	var l models.Label
	query := h.Rebind("SELECT * FROM labels WHERE repo_id = ? AND name = ?;")
	err := h.GetContext(ctx, &l, query, repoID, name)
	return l, db.WrapError(err)
}

// GetLabelsByRepoID implements store.LabelStore.
func (*labelStore) GetLabelsByRepoID(ctx context.Context, h db.Handler, repoID int64) ([]models.Label, error) {
	var labels []models.Label
	query := h.Rebind("SELECT * FROM labels WHERE repo_id = ? ORDER BY name ASC;")
	err := h.SelectContext(ctx, &labels, query, repoID)
	return labels, db.WrapError(err)
}

// GetLabelsByIssueID implements store.LabelStore.
func (*labelStore) GetLabelsByIssueID(ctx context.Context, h db.Handler, issueID int64) ([]models.Label, error) {
	var labels []models.Label
	query := h.Rebind(`SELECT labels.* FROM labels
		JOIN issue_labels ON labels.id = issue_labels.label_id
		WHERE issue_labels.issue_id = ?
		ORDER BY labels.name ASC;`)
	err := h.SelectContext(ctx, &labels, query, issueID)
	return labels, db.WrapError(err)
}

// CreateLabel implements store.LabelStore.
func (*labelStore) CreateLabel(ctx context.Context, h db.Handler, repoID int64, name, color, description string) (int64, error) {
	var id int64
	query := h.Rebind(`INSERT INTO labels (repo_id, name, color, description, created_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP) RETURNING id;`)
	err := h.QueryRowxContext(ctx, query, repoID, name, color, description).Scan(&id)
	return id, db.WrapError(err)
}

// UpdateLabel implements store.LabelStore.
func (*labelStore) UpdateLabel(ctx context.Context, h db.Handler, id, repoID int64, name, color, description string) error {
	query := h.Rebind(`UPDATE labels SET name = ?, color = ?, description = ?
		WHERE id = ? AND repo_id = ?;`)
	_, err := h.ExecContext(ctx, query, name, color, description, id, repoID)
	return db.WrapError(err)
}

// DeleteLabel implements store.LabelStore.
func (*labelStore) DeleteLabel(ctx context.Context, h db.Handler, id, repoID int64) error {
	query := h.Rebind("DELETE FROM labels WHERE id = ? AND repo_id = ?;")
	_, err := h.ExecContext(ctx, query, id, repoID)
	return db.WrapError(err)
}

// AddLabelToIssue implements store.LabelStore.
func (*labelStore) AddLabelToIssue(ctx context.Context, h db.Handler, issueID, labelID int64) error {
	query := h.Rebind("INSERT INTO issue_labels (issue_id, label_id) VALUES (?, ?);")
	_, err := h.ExecContext(ctx, query, issueID, labelID)
	return db.WrapError(err)
}

// RemoveLabelFromIssue implements store.LabelStore.
func (*labelStore) RemoveLabelFromIssue(ctx context.Context, h db.Handler, issueID, labelID int64) error {
	query := h.Rebind("DELETE FROM issue_labels WHERE issue_id = ? AND label_id = ?;")
	_, err := h.ExecContext(ctx, query, issueID, labelID)
	return db.WrapError(err)
}
