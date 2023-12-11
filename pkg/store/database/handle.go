package database

import (
	"context"
	"strings"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/store"
	"github.com/charmbracelet/soft-serve/pkg/utils"
	"github.com/jmoiron/sqlx"
)

type handleStore struct{}

var _ store.HandleStore = &handleStore{}

// CreateHandle implements store.HandleStore.
func (*handleStore) CreateHandle(ctx context.Context, h db.Handler, handle string) (int64, error) {
	handle = strings.ToLower(handle)
	if err := utils.ValidateHandle(handle); err != nil {
		return 0, err
	}

	var id int64
	query := h.Rebind("INSERT INTO handles (handle, updated_at) VALUES (?, CURRENT_TIMESTAMP) RETURNING id;")
	err := h.GetContext(ctx, &id, query, handle)
	return id, db.WrapError(err)
}

// DeleteHandle implements store.HandleStore.
func (*handleStore) DeleteHandle(ctx context.Context, h db.Handler, id int64) error {
	query := h.Rebind("DELETE FROM handles WHERE id = ?;")
	_, err := h.ExecContext(ctx, query, id)
	return db.WrapError(err)
}

// GetHandleByHandle implements store.HandleStore.
func (*handleStore) GetHandleByHandle(ctx context.Context, h db.Handler, handle string) (models.Handle, error) {
	var hl models.Handle
	query := h.Rebind("SELECT * FROM handles WHERE handle = ?;")
	err := h.GetContext(ctx, &hl, query, handle)
	return hl, db.WrapError(err)
}

// GetHandleByID implements store.HandleStore.
func (*handleStore) GetHandleByID(ctx context.Context, h db.Handler, id int64) (models.Handle, error) {
	var hl models.Handle
	query := h.Rebind("SELECT * FROM handles WHERE id = ?;")
	err := h.GetContext(ctx, &hl, query, id)
	return hl, db.WrapError(err)
}

// GetHandleByUserID implements store.HandleStore.
func (*handleStore) GetHandleByUserID(ctx context.Context, h db.Handler, userID int64) (models.Handle, error) {
	var hl models.Handle
	query := h.Rebind("SELECT * FROM handles WHERE id = (SELECT handle_id FROM users WHERE id = ?);")
	err := h.GetContext(ctx, &hl, query, userID)
	return hl, db.WrapError(err)
}

// UpdateHandle implements store.HandleStore.
func (*handleStore) UpdateHandle(ctx context.Context, h db.Handler, id int64, handle string) error {
	handle = strings.ToLower(handle)
	if err := utils.ValidateHandle(handle); err != nil {
		return err
	}
	query := h.Rebind("UPDATE handles SET handle = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;")
	_, err := h.ExecContext(ctx, query, handle, id)
	return db.WrapError(err)
}

// ListHandlesForIDs implements store.HandleStore.
func (*handleStore) ListHandlesForIDs(ctx context.Context, h db.Handler, ids []int64) ([]models.Handle, error) {
	var hls []models.Handle
	if len(ids) == 0 {
		return hls, nil
	}

	query, args, err := sqlx.In("SELECT * FROM handles WHERE id IN (?)", ids)
	if err != nil {
		return nil, db.WrapError(err)
	}

	query = h.Rebind(query)
	err = h.SelectContext(ctx, &hls, query, args...)
	return hls, db.WrapError(err)
}
