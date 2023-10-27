package admin

import (
	"fmt"

	"github.com/charmbracelet/soft-serve/cmd"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/migrate"
	"github.com/spf13/cobra"
)

var (
	// Command is the admin command.
	Command = &cobra.Command{
		Use:   "admin",
		Short: "Administrate the server",
	}

	migrateCmd = &cobra.Command{
		Use:                "migrate",
		Short:              "Migrate the database to the latest version",
		PersistentPreRunE:  cmd.InitBackendContext,
		PersistentPostRunE: cmd.CloseDBContext,
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
		PersistentPreRunE:  cmd.InitBackendContext,
		PersistentPostRunE: cmd.CloseDBContext,
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
		PersistentPreRunE:  cmd.InitBackendContext,
		PersistentPostRunE: cmd.CloseDBContext,
		RunE: func(c *cobra.Command, _ []string) error {
			ctx := c.Context()
			cfg := config.FromContext(ctx)
			be := backend.FromContext(ctx)
			if err := cmd.InitializeHooks(ctx, cfg, be); err != nil {
				return fmt.Errorf("initialize hooks: %w", err)
			}

			return nil
		},
	}
)

func init() {
	Command.AddCommand(
		syncHooksCmd,
		migrateCmd,
		rollbackCmd,
	)
}
