package migrate

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/db"
)

const (
	pushMirrorsName    = "push_mirrors"
	pushMirrorsVersion = 4
)

var pushMirrors = Migration{
	Name:    pushMirrorsName,
	Version: pushMirrorsVersion,
	Migrate: func(ctx context.Context, tx *db.Tx) error {
		return migrateUp(ctx, tx, pushMirrorsVersion, pushMirrorsName)
	},
	Rollback: func(ctx context.Context, tx *db.Tx) error {
		return migrateDown(ctx, tx, pushMirrorsVersion, pushMirrorsName)
	},
}
