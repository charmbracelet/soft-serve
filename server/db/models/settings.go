package models

// Settings represents a settings record.
type Settings struct {
	ID        int64  `db:"id"`
	Key       string `db:"key"`
	Value     string `db:"value"`
	CreatedAt string `db:"created_at"`
	UpdatedAt string `db:"updated_at"`
}
