package backend

import (
	"context"
	"errors"
	"fmt"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/store"
)

// GetIssueAssignees returns all users assigned to an issue.
func (b *Backend) GetIssueAssignees(ctx context.Context, issueID int64) ([]proto.User, error) {
	ms, err := store.FromContext(ctx).GetAssigneesByIssueID(ctx, b.db, issueID)
	if err != nil {
		return nil, err
	}

	users := make([]proto.User, len(ms))
	for i, m := range ms {
		u, err := b.UserByID(ctx, m.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to load assignee user %d: %w", m.ID, err)
		}
		users[i] = u
	}
	return users, nil
}

// AssignUserToIssue assigns a user to an issue. No-op if already assigned.
func (b *Backend) AssignUserToIssue(ctx context.Context, repoName string, issueID int64, username string) error {
	user, err := b.User(ctx, username)
	if err != nil {
		return fmt.Errorf("user %q not found", username)
	}

	issue, err := b.GetIssue(ctx, issueID)
	if err != nil {
		return fmt.Errorf("issue #%d not found", issueID)
	}

	repo, err := b.Repository(ctx, repoName)
	if err != nil {
		return err
	}

	if issue.RepoID() != repo.ID() {
		return fmt.Errorf("issue #%d not found in repository %s", issueID, repoName)
	}

	return b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		err := store.FromContext(ctx).AddAssignee(ctx, tx, issueID, user.ID())
		if errors.Is(err, db.ErrDuplicateKey) {
			return nil // already assigned — idempotent
		}
		return err
	})
}

// UnassignUserFromIssue removes a user from an issue.
func (b *Backend) UnassignUserFromIssue(ctx context.Context, repoName string, issueID int64, username string) error {
	user, err := b.User(ctx, username)
	if err != nil {
		return fmt.Errorf("user %q not found", username)
	}

	issue, err := b.GetIssue(ctx, issueID)
	if err != nil {
		return fmt.Errorf("issue #%d not found", issueID)
	}

	repo, err := b.Repository(ctx, repoName)
	if err != nil {
		return err
	}

	if issue.RepoID() != repo.ID() {
		return fmt.Errorf("issue #%d not found in repository %s", issueID, repoName)
	}

	return b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return store.FromContext(ctx).RemoveAssignee(ctx, tx, issueID, user.ID())
	})
}
