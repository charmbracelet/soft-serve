package database

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/store"
)

type milestoneStore struct{}

var _ store.MilestoneStore = (*milestoneStore)(nil)

// GetMilestoneByID implements store.MilestoneStore.
func (*milestoneStore) GetMilestoneByID(ctx context.Context, h db.Handler, id int64) (models.Milestone, error) {
	var m models.Milestone
	query := h.Rebind("SELECT * FROM milestones WHERE id = ?;")
	err := h.GetContext(ctx, &m, query, id)
	return m, db.WrapError(err)
}

// GetMilestonesByRepoID implements store.MilestoneStore.
func (*milestoneStore) GetMilestonesByRepoID(ctx context.Context, h db.Handler, repoID int64, open bool) ([]models.Milestone, error) {
	var milestones []models.Milestone
	var query string
	if open {
		query = h.Rebind("SELECT * FROM milestones WHERE repo_id = ? AND closed_at IS NULL ORDER BY created_at ASC;")
	} else {
		query = h.Rebind("SELECT * FROM milestones WHERE repo_id = ? AND closed_at IS NOT NULL ORDER BY created_at ASC;")
	}
	err := h.SelectContext(ctx, &milestones, query, repoID)
	return milestones, db.WrapError(err)
}

// CreateMilestone implements store.MilestoneStore.
func (*milestoneStore) CreateMilestone(ctx context.Context, h db.Handler, repoID int64, title, description string) (int64, error) {
	var id int64
	query := h.Rebind(`INSERT INTO milestones (repo_id, title, description, created_at, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP) RETURNING id;`)
	err := h.QueryRowxContext(ctx, query, repoID, title, description).Scan(&id)
	return id, db.WrapError(err)
}

// UpdateMilestone implements store.MilestoneStore.
func (*milestoneStore) UpdateMilestone(ctx context.Context, h db.Handler, id, repoID int64, title, description string) error {
	query := h.Rebind(`UPDATE milestones SET title = ?, description = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND repo_id = ?;`)
	_, err := h.ExecContext(ctx, query, title, description, id, repoID)
	return db.WrapError(err)
}

// CloseMilestone implements store.MilestoneStore.
func (*milestoneStore) CloseMilestone(ctx context.Context, h db.Handler, id, repoID int64) error {
	query := h.Rebind(`UPDATE milestones SET closed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND repo_id = ?;`)
	_, err := h.ExecContext(ctx, query, id, repoID)
	return db.WrapError(err)
}

// ReopenMilestone implements store.MilestoneStore.
func (*milestoneStore) ReopenMilestone(ctx context.Context, h db.Handler, id, repoID int64) error {
	query := h.Rebind(`UPDATE milestones SET closed_at = NULL, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND repo_id = ?;`)
	_, err := h.ExecContext(ctx, query, id, repoID)
	return db.WrapError(err)
}

// DeleteMilestone implements store.MilestoneStore.
func (*milestoneStore) DeleteMilestone(ctx context.Context, h db.Handler, id, repoID int64) error {
	query := h.Rebind("DELETE FROM milestones WHERE id = ? AND repo_id = ?;")
	_, err := h.ExecContext(ctx, query, id, repoID)
	return db.WrapError(err)
}

// SetIssueMilestone implements store.MilestoneStore.
func (*milestoneStore) SetIssueMilestone(ctx context.Context, h db.Handler, issueID, milestoneID int64) error {
	query := h.Rebind("UPDATE issues SET milestone_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;")
	_, err := h.ExecContext(ctx, query, milestoneID, issueID)
	return db.WrapError(err)
}

// UnsetIssueMilestone implements store.MilestoneStore.
func (*milestoneStore) UnsetIssueMilestone(ctx context.Context, h db.Handler, issueID int64) error {
	query := h.Rebind("UPDATE issues SET milestone_id = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = ?;")
	_, err := h.ExecContext(ctx, query, issueID)
	return db.WrapError(err)
}

// GetIssueMilestone implements store.MilestoneStore.
func (*milestoneStore) GetIssueMilestone(ctx context.Context, h db.Handler, issueID int64) (models.Milestone, error) {
	var m models.Milestone
	query := h.Rebind(`SELECT milestones.* FROM milestones
		JOIN issues ON milestones.id = issues.milestone_id
		WHERE issues.id = ?;`)
	err := h.GetContext(ctx, &m, query, issueID)
	return m, db.WrapError(err)
}
