package migrate

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/db"
)

const (
	defaultRepoVisibilityName    = "default_repo_visibility"
	defaultRepoVisibilityVersion = 4
)

var defaultRepoVisibility = Migration{
	Name:    defaultRepoVisibilityName,
	Version: defaultRepoVisibilityVersion,
	Migrate: func(ctx context.Context, tx *db.Tx) error {
		return migrateUp(ctx, tx, defaultRepoVisibilityVersion, defaultRepoVisibilityName)
	},
	Rollback: func(ctx context.Context, tx *db.Tx) error {
		return migrateDown(ctx, tx, defaultRepoVisibilityVersion, defaultRepoVisibilityName)
	},
}
