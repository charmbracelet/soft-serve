package backend

import (
	"context"
	"fmt"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/store"
)

// CreateIssue creates a new issue in a repository.
func (b *Backend) CreateIssue(ctx context.Context, repoName string, userID int64, title, body string) (proto.Issue, error) {
	if title == "" {
		return nil, fmt.Errorf("issue title cannot be empty")
	}

	repo, err := b.Repository(ctx, repoName)
	if err != nil {
		return nil, err
	}

	var issue models.Issue
	err = b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		id, err := b.store.CreateIssue(ctx, tx, repo.ID(), userID, title, body)
		if err != nil {
			return err
		}
		issue, err = b.store.GetIssueByID(ctx, tx, id)
		return err
	})
	if err != nil {
		return nil, err
	}

	return proto.NewIssue(issue), nil
}

// GetIssue retrieves an issue by its ID.
func (b *Backend) GetIssue(ctx context.Context, id int64) (proto.Issue, error) {
	issue, err := b.store.GetIssueByID(ctx, b.db, id)
	if err != nil {
		return nil, err
	}
	return proto.NewIssue(issue), nil
}

// GetIssuesByRepository retrieves issues for a repository filtered by the given IssueFilter.
func (b *Backend) GetIssuesByRepository(ctx context.Context, repoName string, filter store.IssueFilter) ([]proto.Issue, error) {
	repo, err := b.Repository(ctx, repoName)
	if err != nil {
		return nil, err
	}

	issues, err := b.store.GetIssuesByRepoID(ctx, b.db, repo.ID(), filter)
	if err != nil {
		return nil, err
	}

	result := make([]proto.Issue, len(issues))
	for i, issue := range issues {
		result[i] = proto.NewIssue(issue)
	}
	return result, nil
}

// UpdateIssue updates an issue's title and optionally its body.
// A nil body means "do not change the existing body".
func (b *Backend) UpdateIssue(ctx context.Context, id, repoID int64, title string, body *string) error {
	if title == "" {
		return fmt.Errorf("issue title cannot be empty")
	}

	return b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return b.store.UpdateIssue(ctx, tx, id, repoID, title, body)
	})
}

// CloseIssue closes an issue.
func (b *Backend) CloseIssue(ctx context.Context, id, repoID, closedBy int64) error {
	return b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return b.store.CloseIssue(ctx, tx, id, repoID, closedBy)
	})
}

// ReopenIssue reopens a closed issue.
func (b *Backend) ReopenIssue(ctx context.Context, id, repoID int64) error {
	return b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return b.store.ReopenIssue(ctx, tx, id, repoID)
	})
}

// DeleteIssue deletes an issue by its ID.
func (b *Backend) DeleteIssue(ctx context.Context, id, repoID int64) error {
	return b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return b.store.DeleteIssue(ctx, tx, id, repoID)
	})
}

// CountIssues counts issues for a repository filtered by the given IssueFilter.
func (b *Backend) CountIssues(ctx context.Context, repoName string, filter store.IssueFilter) (int64, error) {
	repo, err := b.Repository(ctx, repoName)
	if err != nil {
		return 0, err
	}
	return b.store.CountIssues(ctx, b.db, repo.ID(), filter)
}
