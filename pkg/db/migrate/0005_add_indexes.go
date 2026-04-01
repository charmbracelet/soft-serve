package migrate

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/db"
)

const (
	addIndexesName    = "add_indexes"
	addIndexesVersion = 5
)

var addIndexes = Migration{
	Name:    addIndexesName,
	Version: addIndexesVersion,
	Migrate: func(ctx context.Context, tx *db.Tx) error {
		return migrateUp(ctx, tx, addIndexesVersion, addIndexesName)
	},
	Rollback: func(ctx context.Context, tx *db.Tx) error {
		return migrateDown(ctx, tx, addIndexesVersion, addIndexesName)
	},
}
