package migrate

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/db"
)

const (
	webhooksName    = "webhooks"
	webhooksVersion = 2
)

var webhooks = Migration{
	Name:    webhooksName,
	Version: webhooksVersion,
	Migrate: func(ctx context.Context, h db.Handler) error {
		return migrateUp(ctx, h, webhooksVersion, webhooksName)
	},
	Rollback: func(ctx context.Context, h db.Handler) error {
		return migrateDown(ctx, h, webhooksVersion, webhooksName)
	},
}
