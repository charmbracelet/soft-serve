package models

import (
	"database/sql"
	"time"
)

// AccessToken represents an access token.
type AccessToken struct {
	ID        int64        `db:"id"`
	Name      string       `db:"name"`
	UserID    int64        `db:"user_id"`
	Token     string       `db:"token"`
	ExpiresAt sql.NullTime `db:"expires_at"`
	CreatedAt time.Time    `db:"created_at"`
	UpdatedAt time.Time    `db:"updated_at"`
}
