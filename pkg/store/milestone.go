package store

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
)

// MilestoneStore is an interface for managing milestones.
type MilestoneStore interface {
	// GetMilestoneByID retrieves a milestone by its ID.
	GetMilestoneByID(ctx context.Context, h db.Handler, id int64) (models.Milestone, error)
	// GetMilestonesByRepoID retrieves milestones for a repository.
	// If open is true, only open milestones are returned; otherwise only closed milestones.
	GetMilestonesByRepoID(ctx context.Context, h db.Handler, repoID int64, open bool) ([]models.Milestone, error)
	// CreateMilestone creates a new milestone.
	CreateMilestone(ctx context.Context, h db.Handler, repoID int64, title, description string) (int64, error)
	// UpdateMilestone updates a milestone's title and description.
	UpdateMilestone(ctx context.Context, h db.Handler, id, repoID int64, title, description string) error
	// CloseMilestone closes a milestone.
	CloseMilestone(ctx context.Context, h db.Handler, id, repoID int64) error
	// ReopenMilestone reopens a closed milestone.
	ReopenMilestone(ctx context.Context, h db.Handler, id, repoID int64) error
	// DeleteMilestone deletes a milestone by its ID.
	DeleteMilestone(ctx context.Context, h db.Handler, id, repoID int64) error
	// SetIssueMilestone sets the milestone for an issue.
	SetIssueMilestone(ctx context.Context, h db.Handler, issueID, milestoneID int64) error
	// UnsetIssueMilestone removes the milestone from an issue.
	UnsetIssueMilestone(ctx context.Context, h db.Handler, issueID int64) error
	// GetIssueMilestone retrieves the milestone set on an issue.
	GetIssueMilestone(ctx context.Context, h db.Handler, issueID int64) (models.Milestone, error)
}
