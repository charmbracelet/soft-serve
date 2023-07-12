package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/charmbracelet/soft-serve/server"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/db"
	"github.com/charmbracelet/soft-serve/server/db/migrate"
	"github.com/spf13/cobra"
)

var (
	autoMigrate bool
	rollback    bool

	serveCmd = &cobra.Command{
		Use:   "serve",
		Short: "Start the server",
		Long:  "Start the server",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			cfg := config.DefaultConfig()
			ctx = config.WithContext(ctx, cfg)
			cmd.SetContext(ctx)

			// Create custom hooks directory if it doesn't exist
			customHooksPath := filepath.Join(cfg.DataPath, "hooks")
			if _, err := os.Stat(customHooksPath); err != nil && os.IsNotExist(err) {
				os.MkdirAll(customHooksPath, os.ModePerm) // nolint: errcheck
				// Generate update hook example without executable permissions
				hookPath := filepath.Join(customHooksPath, "update.sample")
				// nolint: gosec
				if err := os.WriteFile(hookPath, []byte(updateHookExample), 0744); err != nil {
					return fmt.Errorf("failed to generate update hook example: %w", err)
				}
			}

			// Create log directory if it doesn't exist
			logPath := filepath.Join(cfg.DataPath, "log")
			if _, err := os.Stat(logPath); err != nil && os.IsNotExist(err) {
				os.MkdirAll(logPath, os.ModePerm) // nolint: errcheck
			}

			db, err := db.Open(ctx, cfg.DB.Driver, cfg.DB.DataSource)
			if err != nil {
				return fmt.Errorf("open database: %w", err)
			}

			if rollback {
				if err := migrate.Rollback(ctx, db); err != nil {
					return fmt.Errorf("rollback error: %w", err)
				}
			} else if autoMigrate {
				if err := migrate.Migrate(ctx, db); err != nil {
					return fmt.Errorf("migration error: %w", err)
				}
			}

			s, err := server.NewServer(ctx)
			if err != nil {
				return fmt.Errorf("start server: %w", err)
			}

			done := make(chan os.Signal, 1)
			lch := make(chan error, 1)
			go func() {
				defer close(lch)
				defer close(done)
				lch <- s.Start()
			}()

			signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
			<-done

			ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			if err := s.Shutdown(ctx); err != nil {
				return err
			}

			// wait for serve to finish
			return <-lch
		},
	}
)

func init() {
	serveCmd.Flags().BoolVarP(&autoMigrate, "auto-migrate", "", false, "automatically run database migrations")
	serveCmd.Flags().BoolVarP(&rollback, "rollback", "", false, "rollback the last database migration")
	rootCmd.AddCommand(serveCmd)
}
