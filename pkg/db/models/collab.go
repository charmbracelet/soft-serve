package models

import (
	"database/sql"
	"time"

	"github.com/charmbracelet/soft-serve/pkg/access"
)

// Collab represents a repository collaborator.
type Collab struct {
	ID          int64              `db:"id"`
	RepoID      int64              `db:"repo_id"`
	UserID      sql.NullInt64      `db:"user_id"`
	TeamID      sql.NullInt64      `db:"team_id"`
	AccessLevel access.AccessLevel `db:"access_level"`
	CreatedAt   time.Time          `db:"created_at"`
	UpdatedAt   time.Time          `db:"updated_at"`
}
