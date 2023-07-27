package migrate

import (
	"context"
	"embed"
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/soft-serve/server/db"
)

//go:embed *.sql
var sqls embed.FS

// Keep this in order of execution, oldest to newest.
var migrations = []Migration{
	createTables,
	createLFSTables,
	passwordTokens,
}

func execMigration(ctx context.Context, tx *db.Tx, version int, name string, down bool) error {
	direction := "up"
	if down {
		direction = "down"
	}

	driverName := tx.DriverName()
	if driverName == "sqlite3" {
		driverName = "sqlite"
	}

	fn := fmt.Sprintf("%04d_%s_%s.%s.sql", version, toSnakeCase(name), driverName, direction)
	sqlstr, err := sqls.ReadFile(fn)
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, string(sqlstr)); err != nil {
		return err
	}

	return nil
}

func migrateUp(ctx context.Context, tx *db.Tx, version int, name string) error {
	return execMigration(ctx, tx, version, name, false)
}

func migrateDown(ctx context.Context, tx *db.Tx, version int, name string) error {
	return execMigration(ctx, tx, version, name, true)
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func toSnakeCase(str string) string {
	str = strings.ReplaceAll(str, "-", "_")
	str = strings.ReplaceAll(str, " ", "_")
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}
