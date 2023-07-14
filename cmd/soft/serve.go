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
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/db"
	"github.com/charmbracelet/soft-serve/server/db/migrate"
	"github.com/charmbracelet/soft-serve/server/hooks"
	"github.com/spf13/cobra"
)

var (
	autoMigrate bool
	rollback    bool
	initHooks   bool

	serveCmd = &cobra.Command{
		Use:                "serve",
		Short:              "Start the server",
		Long:               "Start the server",
		Args:               cobra.NoArgs,
		PersistentPreRunE:  initBackendContext,
		PersistentPostRunE: closeDBContext,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			cfg := config.DefaultConfig()
			if cfg.Exist() {
				if err := cfg.ParseFile(); err != nil {
					return fmt.Errorf("parse config file: %w", err)
				}
			} else {
				if err := cfg.WriteConfig(); err != nil {
					return fmt.Errorf("write config file: %w", err)
				}
			}

			if err := cfg.ParseEnv(); err != nil {
				return fmt.Errorf("parse environment variables: %w", err)
			}

			// Create custom hooks directory if it doesn't exist
			customHooksPath := filepath.Join(cfg.DataPath, "hooks")
			if _, err := os.Stat(customHooksPath); err != nil && os.IsNotExist(err) {
				os.MkdirAll(customHooksPath, os.ModePerm) // nolint: errcheck
				// Generate update hook example without executable permissions
				hookPath := filepath.Join(customHooksPath, "update.sample")
				// nolint: gosec
				if err := os.WriteFile(hookPath, []byte(updateHookExample), 0o744); err != nil {
					return fmt.Errorf("failed to generate update hook example: %w", err)
				}
			}

			// Create log directory if it doesn't exist
			logPath := filepath.Join(cfg.DataPath, "log")
			if _, err := os.Stat(logPath); err != nil && os.IsNotExist(err) {
				os.MkdirAll(logPath, os.ModePerm) // nolint: errcheck
			}

			db := db.FromContext(ctx)
			// TODO: auto migrate by default no flag needed
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

			if initHooks {
				be := backend.FromContext(ctx)
				if err := initializeHooks(ctx, cfg, be); err != nil {
					return fmt.Errorf("initialize hooks: %w", err)
				}
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
	serveCmd.Flags().BoolVarP(&initHooks, "init-hooks", "", false, "initialize the hooks directory and update hooks for all repositories")
	rootCmd.AddCommand(serveCmd)
}

func initializeHooks(ctx context.Context, cfg *config.Config, be *backend.Backend) error {
	repos, err := be.Repositories(ctx)
	if err != nil {
		return err
	}

	for _, repo := range repos {
		if err := hooks.GenerateHooks(ctx, cfg, repo.Name()); err != nil {
			return err
		}
	}

	return nil
}
