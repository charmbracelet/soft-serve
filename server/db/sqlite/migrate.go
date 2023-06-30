package sqlite

import (
	"embed"
	"errors"

	"github.com/charmbracelet/log"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite" // db driver
	_ "github.com/golang-migrate/migrate/v4/source/file"     // file driver
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "modernc.org/sqlite" // sqlite driver
)

//go:embed migrations/*.sql
var migrations embed.FS

// Migrate runs database migrations.
func (s *Sqlite) Migrate(url string) error {
	d, err := iofs.New(migrations, ".")
	if err != nil {
		return err
	}
	m, err := migrate.NewWithSourceInstance("iofs", d, url)
	if err != nil {
		return err
	}
	log.Info("Running migrations...")
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
