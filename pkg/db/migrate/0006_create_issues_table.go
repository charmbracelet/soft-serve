package migrate

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/db"
)

const (
	createIssuesTableName    = "create_issues_table"
	createIssuesTableVersion = 6
)

var createIssuesTable = Migration{
	Name:    createIssuesTableName,
	Version: createIssuesTableVersion,
	Migrate: func(ctx context.Context, tx *db.Tx) error {
		return migrateUp(ctx, tx, createIssuesTableVersion, createIssuesTableName)
	},
	Rollback: func(ctx context.Context, tx *db.Tx) error {
		return migrateDown(ctx, tx, createIssuesTableVersion, createIssuesTableName)
	},
}
