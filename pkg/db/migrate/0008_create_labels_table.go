package migrate

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/db"
)

const (
	createLabelsTableName    = "create_labels_table"
	createLabelsTableVersion = 8
)

var createLabelsTable = Migration{
	Name:    createLabelsTableName,
	Version: createLabelsTableVersion,
	Migrate: func(ctx context.Context, tx *db.Tx) error {
		return migrateUp(ctx, tx, createLabelsTableVersion, createLabelsTableName)
	},
	Rollback: func(ctx context.Context, tx *db.Tx) error {
		return migrateDown(ctx, tx, createLabelsTableVersion, createLabelsTableName)
	},
}
