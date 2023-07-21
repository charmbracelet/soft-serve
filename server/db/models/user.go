package models

import (
	"database/sql"
	"time"
)

// User represents a user.
type User struct {
	ID        int64          `db:"id"`
	Username  string         `db:"username"`
	Admin     bool           `db:"admin"`
	Password  sql.NullString `db:"password"`
	CreatedAt time.Time      `db:"created_at"`
	UpdatedAt time.Time      `db:"updated_at"`
}
