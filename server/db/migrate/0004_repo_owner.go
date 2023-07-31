package migrate

import (
	"context"

	"github.com/charmbracelet/soft-serve/server/db"
)

const (
	repoOwnerName    = "repo owner"
	repoOwnerVersion = 4
)

var repoOwner = Migration{
	Version: repoOwnerVersion,
	Name:    repoOwnerName,
	Migrate: func(ctx context.Context, tx *db.Tx) error {
		return migrateUp(ctx, tx, repoOwnerVersion, repoOwnerName)
	},
	Rollback: func(ctx context.Context, tx *db.Tx) error {
		return migrateDown(ctx, tx, repoOwnerVersion, repoOwnerName)
	},
}
