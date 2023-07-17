package main

import (
	"fmt"

	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/db"
	"github.com/charmbracelet/soft-serve/server/db/migrate"
	"github.com/spf13/cobra"
)

var (
	adminCmd = &cobra.Command{
		Use:   "admin",
		Short: "Administrate the server",
	}

	migrateCmd = &cobra.Command{
		Use:                "migrate",
		Short:              "Migrate the database to the latest version",
		PersistentPreRunE:  initBackendContext,
		PersistentPostRunE: closeDBContext,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			db := db.FromContext(ctx)
			if err := migrate.Migrate(ctx, db); err != nil {
				return fmt.Errorf("migration: %w", err)
			}

			return nil
		},
	}

	rollbackCmd = &cobra.Command{
		Use:                "rollback",
		Short:              "Rollback the database to the previous version",
		PersistentPreRunE:  initBackendContext,
		PersistentPostRunE: closeDBContext,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			db := db.FromContext(ctx)
			if err := migrate.Rollback(ctx, db); err != nil {
				return fmt.Errorf("rollback: %w", err)
			}

			return nil
		},
	}

	syncHooksCmd = &cobra.Command{
		Use:                "sync-hooks",
		Short:              "Update repository hooks",
		PersistentPreRunE:  initBackendContext,
		PersistentPostRunE: closeDBContext,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			cfg := config.FromContext(ctx)
			be := backend.FromContext(ctx)
			if err := initializeHooks(ctx, cfg, be); err != nil {
				return fmt.Errorf("initialize hooks: %w", err)
			}

			return nil
		},
	}
)

func init() {
	adminCmd.AddCommand(
		syncHooksCmd,
		migrateCmd,
		rollbackCmd,
	)
}
