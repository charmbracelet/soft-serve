package models

import (
	"database/sql"
	"time"
)

// Milestone is a database model for a milestone.
type Milestone struct {
	ID          int64        `db:"id"`
	RepoID      int64        `db:"repo_id"`
	Title       string       `db:"title"`
	Description string       `db:"description"`
	DueDate     sql.NullTime `db:"due_date"`
	ClosedAt    sql.NullTime `db:"closed_at"`
	CreatedAt   time.Time    `db:"created_at"`
	UpdatedAt   time.Time    `db:"updated_at"`
}
