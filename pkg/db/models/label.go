package models

import "time"

// Label is a database model for a repository label.
type Label struct {
	ID          int64     `db:"id"`
	RepoID      int64     `db:"repo_id"`
	Name        string    `db:"name"`
	Color       string    `db:"color"`
	Description string    `db:"description"`
	CreatedAt   time.Time `db:"created_at"`
}
