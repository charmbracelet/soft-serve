package models

import "time"

// Collab represents a repository collaborator.
type Collab struct {
	ID        int64     `db:"id"`
	RepoID    int64     `db:"repo_id"`
	UserID    int64     `db:"user_id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
