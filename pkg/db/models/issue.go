package models

import (
	"database/sql"
	"time"
)

// Issue is a database model for an issue.
type Issue struct {
	ID        int64          `db:"id"`
	RepoID    int64          `db:"repo_id"`
	UserID    int64          `db:"user_id"`
	Title     string         `db:"title"`
	Body      sql.NullString `db:"body"`
	Status    string         `db:"status"`
	CreatedAt time.Time      `db:"created_at"`
	UpdatedAt time.Time      `db:"updated_at"`
	ClosedAt  sql.NullTime   `db:"closed_at"`
	ClosedBy  sql.NullInt64  `db:"closed_by"`
}
