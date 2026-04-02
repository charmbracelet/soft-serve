package migrate

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/db"
)

const (
	createAssigneesTableName    = "create_assignees_table"
	createAssigneesTableVersion = 9
)

var createAssigneesTable = Migration{
	Name:    createAssigneesTableName,
	Version: createAssigneesTableVersion,
	Migrate: func(ctx context.Context, tx *db.Tx) error {
		return migrateUp(ctx, tx, createAssigneesTableVersion, createAssigneesTableName)
	},
	Rollback: func(ctx context.Context, tx *db.Tx) error {
		return migrateDown(ctx, tx, createAssigneesTableVersion, createAssigneesTableName)
	},
}
