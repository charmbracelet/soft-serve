// Package database provides database store implementations.
package database

import (
	"context"
	"time"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/store"
)

type accessTokenStore struct{}

var _ store.AccessTokenStore = (*accessTokenStore)(nil)

// CreateAccessToken implements store.AccessTokenStore.
func (s *accessTokenStore) CreateAccessToken(ctx context.Context, h db.Handler, name string, userID int64, token string, expiresAt time.Time) (models.AccessToken, error) {
	queryWithoutExpires := `INSERT INTO access_tokens (name, user_id, token, created_at, updated_at)
	VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP) RETURNING id`
	queryWithExpires := `INSERT INTO access_tokens (name, user_id, token, expires_at, created_at, updated_at)
	VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP) RETURNING id`

	query := queryWithoutExpires
	values := []interface{}{name, userID, token}
	if !expiresAt.IsZero() {
		query = queryWithExpires
		values = append(values, expiresAt.UTC())
	}

	var id int64
	if err := h.GetContext(ctx, &id, h.Rebind(query), values...); err != nil {
		return models.AccessToken{}, err //nolint:wrapcheck
	}

	return s.GetAccessToken(ctx, h, id)
}

// DeleteAccessToken implements store.AccessTokenStore.
func (*accessTokenStore) DeleteAccessToken(ctx context.Context, h db.Handler, id int64) error {
	query := h.Rebind(`DELETE FROM access_tokens WHERE id = ?`)
	_, err := h.ExecContext(ctx, query, id)
	return err //nolint:wrapcheck
}

// DeleteAccessTokenForUser implements store.AccessTokenStore.
func (*accessTokenStore) DeleteAccessTokenForUser(ctx context.Context, h db.Handler, userID int64, id int64) error {
	query := h.Rebind(`DELETE FROM access_tokens WHERE user_id = ? AND id = ?`)
	_, err := h.ExecContext(ctx, query, userID, id)
	return err //nolint:wrapcheck
}

// GetAccessToken implements store.AccessTokenStore.
func (*accessTokenStore) GetAccessToken(ctx context.Context, h db.Handler, id int64) (models.AccessToken, error) {
	query := h.Rebind(`SELECT * FROM access_tokens WHERE id = ?`)
	var m models.AccessToken
	err := h.GetContext(ctx, &m, query, id)
	return m, err //nolint:wrapcheck
}

// GetAccessTokensByUserID implements store.AccessTokenStore.
func (*accessTokenStore) GetAccessTokensByUserID(ctx context.Context, h db.Handler, userID int64) ([]models.AccessToken, error) {
	query := h.Rebind(`SELECT * FROM access_tokens WHERE user_id = ?`)
	var m []models.AccessToken
	err := h.SelectContext(ctx, &m, query, userID)
	return m, err //nolint:wrapcheck
}

// GetAccessTokenByToken implements store.AccessTokenStore.
func (*accessTokenStore) GetAccessTokenByToken(ctx context.Context, h db.Handler, token string) (models.AccessToken, error) {
	query := h.Rebind(`SELECT * FROM access_tokens WHERE token = ?`)
	var m models.AccessToken
	err := h.GetContext(ctx, &m, query, token)
	return m, err //nolint:wrapcheck
}
