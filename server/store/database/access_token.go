package database

import (
	"context"
	"time"

	"github.com/charmbracelet/soft-serve/server/db"
	"github.com/charmbracelet/soft-serve/server/db/models"
	"github.com/charmbracelet/soft-serve/server/store"
)

type accessTokenStore struct{}

var _ store.AccessTokenStore = (*accessTokenStore)(nil)

// CreateAccessToken implements store.AccessTokenStore.
func (s *accessTokenStore) CreateAccessToken(ctx context.Context, h db.Handler, name string, userID int64, token string, expiresAt time.Time) (models.AccessToken, error) {
	queryWithoutExpires := `INSERT INTO access_tokens (name, user_id, token, created_at, updated_at)
	VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`
	queryWithExpires := `INSERT INTO access_tokens (name, user_id, token, expires_at, created_at, updated_at)
	VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`

	query := queryWithoutExpires
	values := []interface{}{name, userID, token}
	if !expiresAt.IsZero() {
		query = queryWithExpires
		values = append(values, expiresAt)
	}

	result, err := h.ExecContext(ctx, query, values...)
	if err != nil {
		return models.AccessToken{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return models.AccessToken{}, err
	}

	return s.GetAccessToken(ctx, h, id)
}

// DeleteAccessToken implements store.AccessTokenStore.
func (*accessTokenStore) DeleteAccessToken(ctx context.Context, h db.Handler, id int64) error {
	query := h.Rebind(`DELETE FROM access_tokens WHERE id = ?`)
	_, err := h.ExecContext(ctx, query, id)
	return err
}

// DeleteAccessTokenForUser implements store.AccessTokenStore.
func (*accessTokenStore) DeleteAccessTokenForUser(ctx context.Context, h db.Handler, userID int64, id int64) error {
	query := h.Rebind(`DELETE FROM access_tokens WHERE user_id = ? AND id = ?`)
	_, err := h.ExecContext(ctx, query, userID, id)
	return err
}

// GetAccessToken implements store.AccessTokenStore.
func (*accessTokenStore) GetAccessToken(ctx context.Context, h db.Handler, id int64) (models.AccessToken, error) {
	query := h.Rebind(`SELECT * FROM access_tokens WHERE id = ?`)
	var m models.AccessToken
	err := h.GetContext(ctx, &m, query, id)
	return m, err
}

// GetAccessTokensByUserID implements store.AccessTokenStore.
func (*accessTokenStore) GetAccessTokensByUserID(ctx context.Context, h db.Handler, userID int64) ([]models.AccessToken, error) {
	query := h.Rebind(`SELECT * FROM access_tokens WHERE user_id = ?`)
	var m []models.AccessToken
	err := h.SelectContext(ctx, &m, query, userID)
	return m, err
}

// GetAccessTokenByToken implements store.AccessTokenStore.
func (*accessTokenStore) GetAccessTokenByToken(ctx context.Context, h db.Handler, token string) (models.AccessToken, error) {
	query := h.Rebind(`SELECT * FROM access_tokens WHERE token = ?`)
	var m models.AccessToken
	err := h.GetContext(ctx, &m, query, token)
	return m, err
}
