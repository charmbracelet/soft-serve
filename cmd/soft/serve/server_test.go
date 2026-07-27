package serve

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"charm.land/log/v2"
	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/migrate"
	"github.com/charmbracelet/soft-serve/pkg/store"
	"github.com/charmbracelet/soft-serve/pkg/store/database"
	"github.com/matryer/is"
	_ "modernc.org/sqlite"
)

// newTestBackend returns a Backend backed by a real, freshly migrated
// SQLite database, along with the *config.Config it was constructed with,
// so tests can set AnonAccess/AllowKeyless overrides directly.
func newTestBackend(t *testing.T) (*backend.Backend, *config.Config) {
	t.Helper()
	is := is.New(t)
	ctx := context.Background()

	dp := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.DataPath = dp
	cfg.DB.Driver = "sqlite"
	cfg.DB.DataSource = dp + "/test.db"

	ctx = config.WithContext(ctx, cfg)
	dbx, err := db.Open(ctx, cfg.DB.Driver, cfg.DB.DataSource)
	is.NoErr(err)
	t.Cleanup(func() { dbx.Close() }) //nolint:errcheck

	is.NoErr(migrate.Migrate(ctx, dbx))
	dbstore := database.New(ctx, dbx)
	ctx = store.WithContext(ctx, dbstore)
	be := backend.New(ctx, cfg, dbx, dbstore)

	return be, cfg
}

func TestWarnIfAnonAdminAccess(t *testing.T) {
	cases := []struct {
		name         string
		allowKeyless bool
		anonAccess   access.AccessLevel
		wantWarning  bool
	}{
		{"defaults: keyless disabled, read-only anon", false, access.ReadOnlyAccess, false},
		{"keyless allowed but anon access below admin", true, access.ReadWriteAccess, false},
		{"admin anon access but keyless disallowed", false, access.AdminAccess, false},
		{"keyless allowed with admin anon access: the dangerous combo", true, access.AdminAccess, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			is := is.New(t)
			be, cfg := newTestBackend(t)
			ctx := context.Background()

			allow := c.allowKeyless
			cfg.AllowKeyless = &allow
			cfg.AnonAccess = c.anonAccess.String()

			var buf bytes.Buffer
			logger := log.New(&buf)

			warnIfAnonAdminAccess(ctx, be, logger)

			gotWarning := strings.Contains(buf.String(), "WARNING")
			is.Equal(gotWarning, c.wantWarning)
		})
	}
}
