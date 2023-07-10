package models

// PublicKey represents a public key.
type PublicKey struct {
	ID        int64  `db:"id"`
	UserID    int64  `db:"user_id"`
	PublicKey string `db:"public_key"`
	CreatedAt string `db:"created_at"`
	UpdatedAt string `db:"updated_at"`
}
