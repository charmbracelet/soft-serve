package backend

import (
	"context"
	"fmt"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/proto"
)

// AddIssueComment adds a comment to an issue.
func (b *Backend) AddIssueComment(ctx context.Context, issueID, userID int64, body string) (proto.IssueComment, error) {
	if body == "" {
		return nil, fmt.Errorf("comment body cannot be empty")
	}

	// Verify the issue exists before commenting.
	if _, err := b.store.GetIssueByID(ctx, b.db, issueID); err != nil {
		return nil, err
	}

	var id int64
	err := b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		id, err = b.store.CreateIssueComment(ctx, tx, issueID, userID, body)
		return err
	})
	if err != nil {
		return nil, err
	}

	c, err := b.store.GetIssueCommentByID(ctx, b.db, id)
	if err != nil {
		return nil, err
	}
	return proto.NewIssueComment(c), nil
}

// GetIssueComment retrieves a single comment by ID.
func (b *Backend) GetIssueComment(ctx context.Context, id int64) (proto.IssueComment, error) {
	c, err := b.store.GetIssueCommentByID(ctx, b.db, id)
	if err != nil {
		return nil, err
	}
	return proto.NewIssueComment(c), nil
}

// GetIssueComments retrieves all comments for an issue.
func (b *Backend) GetIssueComments(ctx context.Context, issueID int64) ([]proto.IssueComment, error) {
	// Verify the issue exists.
	if _, err := b.store.GetIssueByID(ctx, b.db, issueID); err != nil {
		return nil, err
	}

	comments, err := b.store.GetCommentsByIssueID(ctx, b.db, issueID)
	if err != nil {
		return nil, err
	}

	result := make([]proto.IssueComment, len(comments))
	for i, c := range comments {
		result[i] = proto.NewIssueComment(c)
	}
	return result, nil
}

// UpdateIssueComment updates the body of a comment.
func (b *Backend) UpdateIssueComment(ctx context.Context, id int64, body string) error {
	if body == "" {
		return fmt.Errorf("comment body cannot be empty")
	}

	return b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return b.store.UpdateIssueComment(ctx, tx, id, body)
	})
}

// DeleteIssueComment deletes a comment by ID.
func (b *Backend) DeleteIssueComment(ctx context.Context, id int64) error {
	return b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return b.store.DeleteIssueComment(ctx, tx, id)
	})
}
