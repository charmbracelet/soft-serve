package models

import "time"

// Handle represents a name handle.
type Handle struct {
	ID        int64     `db:"id"`
	Handle    string    `db:"handle"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
