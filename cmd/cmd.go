// Package cmd provides common command functionality for soft-serve.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/hooks"
	"github.com/charmbracelet/soft-serve/pkg/store"
	"github.com/charmbracelet/soft-serve/pkg/store/database"
	"github.com/spf13/cobra"
)

// InitBackendContext initializes the backend context.
func InitBackendContext(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	cfg := config.FromContext(ctx)
	if _, err := os.Stat(cfg.DataPath); errors.Is(err, fs.ErrNotExist) {
		if err := os.MkdirAll(cfg.DataPath, os.ModePerm); err != nil { //nolint:gosec
			return fmt.Errorf("create data directory: %w", err)
		}
	}
	dbx, err := db.Open(ctx, cfg.DB.Driver, cfg.DB.DataSource)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	ctx = db.WithContext(ctx, dbx)
	dbstore := database.New(ctx, dbx)
	ctx = store.WithContext(ctx, dbstore)
	be := backend.New(ctx, cfg, dbx, dbstore)
	ctx = backend.WithContext(ctx, be)

	cmd.SetContext(ctx)

	return nil
}

// CloseDBContext closes the database context.
func CloseDBContext(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	dbx := db.FromContext(ctx)
	if dbx != nil {
		if err := dbx.Close(); err != nil {
			return fmt.Errorf("close database: %w", err)
		}
	}

	return nil
}

// InitializeHooks initializes the hooks.
func InitializeHooks(ctx context.Context, cfg *config.Config, be *backend.Backend) error {
	repos, err := be.Repositories(ctx)
	if err != nil {
		return fmt.Errorf("failed to get repositories: %w", err)
	}

	for _, repo := range repos {
		if err := hooks.GenerateHooks(ctx, cfg, repo.Name()); err != nil {
			return fmt.Errorf("failed to generate hooks for repo %s: %w", repo.Name(), err)
		}
	}

	return nil
}
