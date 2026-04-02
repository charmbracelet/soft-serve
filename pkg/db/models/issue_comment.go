package models

import "time"

// IssueComment is a database model for an issue comment.
type IssueComment struct {
	ID        int64     `db:"id"`
	IssueID   int64     `db:"issue_id"`
	UserID    int64     `db:"user_id"`
	Body      string    `db:"body"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
