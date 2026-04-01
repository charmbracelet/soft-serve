package migrate

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/db"
)

const (
	createIssueCommentsTableName    = "create_issue_comments_table"
	createIssueCommentsTableVersion = 7
)

var createIssueCommentsTable = Migration{
	Name:    createIssueCommentsTableName,
	Version: createIssueCommentsTableVersion,
	Migrate: func(ctx context.Context, tx *db.Tx) error {
		return migrateUp(ctx, tx, createIssueCommentsTableVersion, createIssueCommentsTableName)
	},
	Rollback: func(ctx context.Context, tx *db.Tx) error {
		return migrateDown(ctx, tx, createIssueCommentsTableVersion, createIssueCommentsTableName)
	},
}
