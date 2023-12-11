package migrate

import (
	"context"
	"strings"

	"github.com/charmbracelet/soft-serve/pkg/db"
)

const (
	createOrgsTeamsName    = "create_orgs_teams"
	createOrgsTeamsVersion = 4
)

var createOrgsTeams = Migration{
	Name:    createOrgsTeamsName,
	Version: createOrgsTeamsVersion,
	PreMigrate: func(ctx context.Context, h db.Handler) error {
		if strings.HasPrefix(h.DriverName(), "sqlite") {
			if _, err := h.ExecContext(ctx, "PRAGMA foreign_keys = OFF;"); err != nil {
				return err
			}
			if _, err := h.ExecContext(ctx, "PRAGMA legacy_alter_table = ON;"); err != nil {
				return err
			}
		}
		return nil
	},
	PostMigrate: func(ctx context.Context, h db.Handler) error {
		if strings.HasPrefix(h.DriverName(), "sqlite") {
			if _, err := h.ExecContext(ctx, "PRAGMA foreign_keys = ON;"); err != nil {
				return err
			}
			if _, err := h.ExecContext(ctx, "PRAGMA legacy_alter_table = OFF;"); err != nil {
				return err
			}
		}
		return nil
	},
	Migrate: func(ctx context.Context, h db.Handler) error {
		return migrateUp(ctx, h, createOrgsTeamsVersion, createOrgsTeamsName)
	},
	Rollback: func(ctx context.Context, h db.Handler) error {
		return migrateDown(ctx, h, createOrgsTeamsVersion, createOrgsTeamsName)
	},
}
