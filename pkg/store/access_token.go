// Package store provides data store functionality.
package store

import (
	"context"
	"time"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
)

// AccessTokenStore is an interface for managing access tokens.
type AccessTokenStore interface {
	GetAccessToken(ctx context.Context, h db.Handler, id int64) (models.AccessToken, error)
	GetAccessTokenByToken(ctx context.Context, h db.Handler, token string) (models.AccessToken, error)
	GetAccessTokensByUserID(ctx context.Context, h db.Handler, userID int64) ([]models.AccessToken, error)
	CreateAccessToken(ctx context.Context, h db.Handler, name string, userID int64, token string, expiresAt time.Time) (models.AccessToken, error)
	DeleteAccessToken(ctx context.Context, h db.Handler, id int64) error
	DeleteAccessTokenForUser(ctx context.Context, h db.Handler, userID int64, id int64) error
}
