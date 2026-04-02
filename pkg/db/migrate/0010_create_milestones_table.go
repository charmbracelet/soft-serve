package migrate

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/db"
)

const (
	createMilestonesTableName    = "create_milestones_table"
	createMilestonesTableVersion = 10
)

var createMilestonesTable = Migration{
	Name:    createMilestonesTableName,
	Version: createMilestonesTableVersion,
	Migrate: func(ctx context.Context, tx *db.Tx) error {
		return migrateUp(ctx, tx, createMilestonesTableVersion, createMilestonesTableName)
	},
	Rollback: func(ctx context.Context, tx *db.Tx) error {
		return migrateDown(ctx, tx, createMilestonesTableVersion, createMilestonesTableName)
	},
}
