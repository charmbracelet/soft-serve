package backend

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/proto"
)

// CreateIssue creates a new issue in a repository.
func (b *Backend) CreateIssue(ctx context.Context, repoName string, userID int64, title, body string) (proto.Issue, error) {
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
	var issue models.Issue
	err := b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		issue, err = b.store.GetIssueByID(ctx, tx, id)
		return err
	})
	if err != nil {
		return nil, err
	}

	return proto.NewIssue(issue), nil
}

// GetIssuesByRepository retrieves all issues for a repository.
func (b *Backend) GetIssuesByRepository(ctx context.Context, repoName string, status string) ([]proto.Issue, error) {
	repo, err := b.Repository(ctx, repoName)
	if err != nil {
		return nil, err
	}

	var issues []models.Issue
	err = b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		issues, err = b.store.GetIssuesByRepoID(ctx, tx, repo.ID(), status)
		return err
	})
	if err != nil {
		return nil, err
	}

	result := make([]proto.Issue, len(issues))
	for i, issue := range issues {
		result[i] = proto.NewIssue(issue)
	}
	return result, nil
}

// UpdateIssue updates an issue's title and body.
func (b *Backend) UpdateIssue(ctx context.Context, id int64, title, body string) error {
	return b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return b.store.UpdateIssue(ctx, tx, id, title, body)
	})
}

// CloseIssue closes an issue.
func (b *Backend) CloseIssue(ctx context.Context, id, closedBy int64) error {
	return b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return b.store.CloseIssue(ctx, tx, id, closedBy)
	})
}

// ReopenIssue reopens a closed issue.
func (b *Backend) ReopenIssue(ctx context.Context, id int64) error {
	return b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return b.store.ReopenIssue(ctx, tx, id)
	})
}

// DeleteIssue deletes an issue by its ID.
func (b *Backend) DeleteIssue(ctx context.Context, id int64) error {
	return b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return b.store.DeleteIssue(ctx, tx, id)
	})
}

// CountIssues counts issues for a repository.
func (b *Backend) CountIssues(ctx context.Context, repoName string, status string) (int64, error) {
	repo, err := b.Repository(ctx, repoName)
	if err != nil {
		return 0, err
	}

	var count int64
	err = b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		count, err = b.store.CountIssues(ctx, tx, repo.ID(), status)
		return err
	})
	return count, err
}
