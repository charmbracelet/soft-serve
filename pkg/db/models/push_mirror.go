package models

import "time"

// PushMirror represents a push mirror for a repository.
type PushMirror struct {
	ID        int64     `db:"id"`
	RepoID    int64     `db:"repo_id"`
	Name      string    `db:"name"`
	RemoteURL string    `db:"remote_url"`
	Enabled   bool      `db:"enabled"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
