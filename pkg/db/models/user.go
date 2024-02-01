package models

import (
	"database/sql"
	"time"
)

// User represents a user.
type User struct {
	ID        int64          `db:"id"`
	Name      sql.NullString `db:"name"`
	Admin     bool           `db:"admin"`
	Password  sql.NullString `db:"password"`
	HandleID  int64          `db:"handle_id"`
	CreatedAt time.Time      `db:"created_at"`
	UpdatedAt time.Time      `db:"updated_at"`
}

// UserEmail represents a user's email address.
type UserEmail struct {
	ID        int64     `db:"id"`
	UserID    int64     `db:"user_id"`
	Email     string    `db:"email"`
	IsPrimary bool      `db:"is_primary"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
