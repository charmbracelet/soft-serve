package proto

import "time"

// AccessToken represents an access token.
type AccessToken struct {
	ID        int64
	Name      string
	UserID    int64
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
}
