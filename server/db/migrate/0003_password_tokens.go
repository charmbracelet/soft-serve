package migrate

import (
	"context"

	"github.com/charmbracelet/soft-serve/server/db"
)

const (
	passwordTokensName    = "password tokens"
	passwordTokensVersion = 3
)

var passwordTokens = Migration{
	Version: passwordTokensVersion,
	Name:    passwordTokensName,
	Migrate: func(ctx context.Context, tx *db.Tx) error {
		return migrateUp(ctx, tx, passwordTokensVersion, passwordTokensName)
	},
	Rollback: func(ctx context.Context, tx *db.Tx) error {
		return migrateDown(ctx, tx, passwordTokensVersion, passwordTokensName)
	},
}
