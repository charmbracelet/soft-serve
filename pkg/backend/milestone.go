package backend

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/store"
)

// CreateMilestone creates a new milestone for a repository.
// Title must be non-empty.
func (b *Backend) CreateMilestone(ctx context.Context, repoName, title, description string) (proto.Milestone, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, fmt.Errorf("milestone title cannot be empty")
	}

	repo, err := b.Repository(ctx, repoName)
	if err != nil {
		return nil, err
	}

	var id int64
	if err := b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		id, err = store.FromContext(ctx).CreateMilestone(ctx, tx, repo.ID(), title, description)
		return err
	}); err != nil {
		return nil, err
	}

	return b.getMilestoneByID(ctx, id)
}

// GetMilestone retrieves a milestone by ID from a repository.
func (b *Backend) GetMilestone(ctx context.Context, repoName string, id int64) (proto.Milestone, error) {
	repo, err := b.Repository(ctx, repoName)
	if err != nil {
		return nil, err
	}

	m, err := store.FromContext(ctx).GetMilestoneByID(ctx, b.db, id)
	if err != nil {
		return nil, fmt.Errorf("milestone #%d not found in repository %s", id, repoName)
	}
	if m.RepoID != repo.ID() {
		return nil, fmt.Errorf("milestone #%d not found in repository %s", id, repoName)
	}

	return proto.NewMilestone(m), nil
}

// ListMilestones returns milestones for a repository.
// If open is true, only open milestones are returned; otherwise only closed milestones.
func (b *Backend) ListMilestones(ctx context.Context, repoName string, open bool) ([]proto.Milestone, error) {
	repo, err := b.Repository(ctx, repoName)
	if err != nil {
		return nil, err
	}

	ms, err := store.FromContext(ctx).GetMilestonesByRepoID(ctx, b.db, repo.ID(), open)
	if err != nil {
		return nil, err
	}

	milestones := make([]proto.Milestone, len(ms))
	for i, m := range ms {
		milestones[i] = proto.NewMilestone(m)
	}
	return milestones, nil
}

// UpdateMilestone updates a milestone's title and description.
func (b *Backend) UpdateMilestone(ctx context.Context, repoName string, id int64, title, description string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return fmt.Errorf("milestone title cannot be empty")
	}

	repo, err := b.Repository(ctx, repoName)
	if err != nil {
		return err
	}

	return b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return store.FromContext(ctx).UpdateMilestone(ctx, tx, id, repo.ID(), title, description)
	})
}

// CloseMilestone closes a milestone.
func (b *Backend) CloseMilestone(ctx context.Context, repoName string, id int64) error {
	repo, err := b.Repository(ctx, repoName)
	if err != nil {
		return err
	}

	return b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return store.FromContext(ctx).CloseMilestone(ctx, tx, id, repo.ID())
	})
}

// ReopenMilestone reopens a closed milestone.
func (b *Backend) ReopenMilestone(ctx context.Context, repoName string, id int64) error {
	repo, err := b.Repository(ctx, repoName)
	if err != nil {
		return err
	}

	return b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return store.FromContext(ctx).ReopenMilestone(ctx, tx, id, repo.ID())
	})
}

// DeleteMilestone deletes a milestone from a repository.
func (b *Backend) DeleteMilestone(ctx context.Context, repoName string, id int64) error {
	repo, err := b.Repository(ctx, repoName)
	if err != nil {
		return err
	}

	return b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return store.FromContext(ctx).DeleteMilestone(ctx, tx, id, repo.ID())
	})
}

// SetIssueMilestone sets the milestone for an issue.
func (b *Backend) SetIssueMilestone(ctx context.Context, issueID, milestoneID int64) error {
	return b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return store.FromContext(ctx).SetIssueMilestone(ctx, tx, issueID, milestoneID)
	})
}

// UnsetIssueMilestone removes the milestone from an issue.
func (b *Backend) UnsetIssueMilestone(ctx context.Context, issueID int64) error {
	return b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return store.FromContext(ctx).UnsetIssueMilestone(ctx, tx, issueID)
	})
}

// GetIssueMilestone retrieves the milestone set on an issue.
// Returns nil, nil if the issue has no milestone.
func (b *Backend) GetIssueMilestone(ctx context.Context, issueID int64) (proto.Milestone, error) {
	m, err := store.FromContext(ctx).GetIssueMilestone(ctx, b.db, issueID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return proto.NewMilestone(m), nil
}

// getMilestoneByID is a helper for looking up a milestone by its database ID.
func (b *Backend) getMilestoneByID(ctx context.Context, id int64) (proto.Milestone, error) {
	m, err := store.FromContext(ctx).GetMilestoneByID(ctx, b.db, id)
	if err != nil {
		return nil, err
	}
	return proto.NewMilestone(m), nil
}
