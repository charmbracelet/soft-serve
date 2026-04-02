package proto

import (
	"time"

	"github.com/charmbracelet/soft-serve/pkg/db/models"
)

type issueComment struct {
	model models.IssueComment
}

// NewIssueComment wraps a models.IssueComment in the IssueComment interface.
func NewIssueComment(m models.IssueComment) IssueComment {
	return &issueComment{model: m}
}

func (c *issueComment) ID() int64        { return c.model.ID }
func (c *issueComment) IssueID() int64   { return c.model.IssueID }
func (c *issueComment) UserID() int64    { return c.model.UserID }
func (c *issueComment) Body() string     { return c.model.Body }
func (c *issueComment) CreatedAt() time.Time { return c.model.CreatedAt }
func (c *issueComment) UpdatedAt() time.Time { return c.model.UpdatedAt }
