package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/hooks"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/sshutils"
	"github.com/charmbracelet/soft-serve/pkg/store"
	"github.com/charmbracelet/soft-serve/pkg/store/database"
	"github.com/spf13/cobra"
)

// InitBackendContext initializes the backend context.
// When a public-key is provided via the "SOFT_SERVE_PUBLIC_KEY" environment
// variable, it will be used to try to find the corresponding user in the
// database and set the user in the context.
func InitBackendContext(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	cfg := config.FromContext(ctx)
	if _, err := os.Stat(cfg.DataPath); errors.Is(err, fs.ErrNotExist) {
		if err := os.MkdirAll(cfg.DataPath, os.ModePerm); err != nil {
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
	be := backend.New(ctx, cfg, dbx)
	ctx = backend.WithContext(ctx, be)

	// Store user in context if public key is provided
	// via environment variable.
	if ak, ok := os.LookupEnv("SOFT_SERVE_PUBLIC_KEY"); ok {
		pk, _, err := sshutils.ParseAuthorizedKey(ak)
		if err == nil && pk != nil {
			user, err := be.UserByPublicKey(ctx, pk)
			if err == nil && user != nil {
				ctx = proto.WithUserContext(ctx, user)
			}
		}
	}

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
		return err
	}

	for _, repo := range repos {
		if err := hooks.GenerateHooks(ctx, cfg, repo.Name()); err != nil {
			return err
		}
	}

	return nil
}

// CheckUserHasAccess checks if the user in context has access to the repository.
// If there is no user in context, it will skip the check and return true.
// It won't skip this check if the "strict" flag is set to true.
func CheckUserHasAccess(cmd *cobra.Command, repo string, level access.AccessLevel) bool {
	ctx := cmd.Context()
	isStrict := cmd.Flag("strict").Value.String() == "true"
	if _, ok := os.LookupEnv("SOFT_SERVE_PUBLIC_KEY"); !ok && isStrict {
		return false
	}

	user := proto.UserFromContext(ctx)
	be := backend.FromContext(ctx)
	al := be.AccessLevelForUser(ctx, repo, user)
	return al >= level
}
