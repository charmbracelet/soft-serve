package database

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/store"
)

type pushMirrorStore struct{}

var _ store.PushMirrorStore = (*pushMirrorStore)(nil)

// CreatePushMirror creates a push mirror for a repository.
func (*pushMirrorStore) CreatePushMirror(ctx context.Context, h db.Handler, repoID int64, name, remoteURL string) error {
	query := h.Rebind(`INSERT INTO push_mirrors (repo_id, name, remote_url) VALUES (?, ?, ?);`)
	_, err := h.ExecContext(ctx, query, repoID, name, remoteURL)
	return db.WrapError(err)
}

// DeletePushMirror deletes a push mirror by repo and name.
func (*pushMirrorStore) DeletePushMirror(ctx context.Context, h db.Handler, repoID int64, name string) error {
	query := h.Rebind(`DELETE FROM push_mirrors WHERE repo_id = ? AND name = ?;`)
	_, err := h.ExecContext(ctx, query, repoID, name)
	return db.WrapError(err)
}

// GetPushMirrorsByRepoID returns all push mirrors for a repository.
func (*pushMirrorStore) GetPushMirrorsByRepoID(ctx context.Context, h db.Handler, repoID int64) ([]models.PushMirror, error) {
	var mirrors []models.PushMirror
	query := h.Rebind(`SELECT id, repo_id, name, remote_url, enabled, created_at, updated_at FROM push_mirrors WHERE repo_id = ? ORDER BY name;`)
	err := h.SelectContext(ctx, &mirrors, query, repoID)
	return mirrors, db.WrapError(err)
}

// SetPushMirrorEnabled enables or disables a push mirror.
func (*pushMirrorStore) SetPushMirrorEnabled(ctx context.Context, h db.Handler, repoID int64, name string, enabled bool) error {
	query := h.Rebind(`UPDATE push_mirrors SET enabled = ?, updated_at = CURRENT_TIMESTAMP WHERE repo_id = ? AND name = ?;`)
	_, err := h.ExecContext(ctx, query, enabled, repoID, name)
	return db.WrapError(err)
}
