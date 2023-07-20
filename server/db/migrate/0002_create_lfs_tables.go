package migrate

import (
	"context"

	"github.com/charmbracelet/soft-serve/server/db"
)

const (
	createLFSTablesName    = "create lfs tables"
	createLFSTablesVersion = 2
)

var createLFSTables = Migration{
	Version: createLFSTablesVersion,
	Name:    createLFSTablesName,
	Migrate: func(ctx context.Context, tx *db.Tx) error {
		return migrateUp(ctx, tx, createLFSTablesVersion, createLFSTablesName)
	},
	Rollback: func(ctx context.Context, tx *db.Tx) error {
		return migrateDown(ctx, tx, createLFSTablesVersion, createLFSTablesName)
	},
}
