package backend

import (
	"context"
	"errors"
	"time"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/proto"
)

// CreateAccessToken creates an access token for user.
func (b *Backend) CreateAccessToken(ctx context.Context, user proto.User, name string, expiresAt time.Time) (string, error) {
	token := GenerateToken()
	tokenHash := HashToken(token)

	if err := b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		_, err := b.store.CreateAccessToken(ctx, tx, name, user.ID(), tokenHash, expiresAt)
		if err != nil {
			return db.WrapError(err)
		}

		return nil
	}); err != nil {
		return "", err
	}

	return token, nil
}

// DeleteAccessToken deletes an access token for a user.
func (b *Backend) DeleteAccessToken(ctx context.Context, user proto.User, id int64) error {
	err := b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		_, err := b.store.GetAccessToken(ctx, tx, id)
		if err != nil {
			return db.WrapError(err)
		}

		if err := b.store.DeleteAccessTokenForUser(ctx, tx, user.ID(), id); err != nil {
			return db.WrapError(err)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			return proto.ErrTokenNotFound
		}
		return err
	}

	return nil
}

// ListAccessTokens lists access tokens for a user.
func (b *Backend) ListAccessTokens(ctx context.Context, user proto.User) ([]proto.AccessToken, error) {
	accessTokens, err := b.store.GetAccessTokensByUserID(ctx, b.db, user.ID())
	if err != nil {
		return nil, db.WrapError(err)
	}

		tokens := make([]proto.AccessToken, 0, len(accessTokens))
	for _, t := range accessTokens {
		token := proto.AccessToken{
			ID:        t.ID,
			Name:      t.Name,
			TokenHash: t.Token,
			UserID:    t.UserID,
			CreatedAt: t.CreatedAt,
		}
		if t.ExpiresAt.Valid {
			token.ExpiresAt = t.ExpiresAt.Time
		}

		tokens = append(tokens, token)
	}

	return tokens, nil
}
