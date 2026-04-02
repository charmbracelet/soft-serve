package store

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
)

// DefaultIssueLimit is the default number of issues returned per page.
const DefaultIssueLimit = 25

// IssueFilter controls which issues are returned by GetIssuesByRepoID / CountIssues.
type IssueFilter struct {
	Status      string // "open", "closed", "all", "" (same as "all")
	Search      string // keyword matched against title and body (LIKE)
	LabelName   string // filter by label name (join with labels table)
	MilestoneID int64  // filter by milestone ID (0 = no filter)
	Page        int    // 1-based page number; <=1 means page 1
	Limit       int    // results per page; <=0 means DefaultIssueLimit
}

// IssueStore is an interface for managing issues.
type IssueStore interface {
	// GetIssueByID retrieves an issue by its ID.
	GetIssueByID(ctx context.Context, h db.Handler, id int64) (models.Issue, error)
	// GetIssuesByRepoID retrieves issues for a repository filtered by the given IssueFilter.
	GetIssuesByRepoID(ctx context.Context, h db.Handler, repoID int64, filter IssueFilter) ([]models.Issue, error)
	// CreateIssue creates a new issue.
	CreateIssue(ctx context.Context, h db.Handler, repoID, userID int64, title, body string) (int64, error)
	// UpdateIssue updates an issue's title and optionally its body.
	// A nil body means "do not change the body".
	UpdateIssue(ctx context.Context, h db.Handler, id, repoID int64, title string, body *string) error
	// CloseIssue closes an issue.
	CloseIssue(ctx context.Context, h db.Handler, id, repoID, closedBy int64) error
	// ReopenIssue reopens a closed issue.
	ReopenIssue(ctx context.Context, h db.Handler, id, repoID int64) error
	// DeleteIssue deletes an issue by its ID.
	DeleteIssue(ctx context.Context, h db.Handler, id, repoID int64) error
	// CountIssues counts issues for a repository filtered by the given IssueFilter.
	CountIssues(ctx context.Context, h db.Handler, repoID int64, filter IssueFilter) (int64, error)
}
