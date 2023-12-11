package models

import (
	"time"
)

// Team represents a team in an organization.
type Team struct {
	ID             int64     `db:"id"`
	Name           string    `db:"name"`
	OrganizationID int64     `db:"org_id"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

// TeamMember represents a member of a team.
type TeamMember struct {
	ID        int64     `db:"id"`
	TeamID    int64     `db:"team_id"`
	UserID    int64     `db:"user_id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
