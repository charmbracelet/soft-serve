package models

import (
	"database/sql"
	"time"
)

// Organization represents an organization in the system.
type Organization struct {
	ID           int64          `db:"id"`
	Name         sql.NullString `db:"name"`
	ContactEmail string         `db:"contact_email"`
	CreatedAt    time.Time      `db:"created_at"`
	UpdatedAt    time.Time      `db:"updated_at"`
	Handle       Handle         `db:"handle"`
}

// OrganizationMember represents a member of an organization.
type OrganizationMember struct {
	ID             int64     `db:"id"`
	OrganizationID int64     `db:"org_id"`
	UserID         int64     `db:"user_id"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}
