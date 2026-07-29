package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/migrate"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/store"
	"github.com/charmbracelet/soft-serve/pkg/store/database"
	"github.com/matryer/is"
	_ "modernc.org/sqlite"
)

// newSettingsTestContext returns a context wired with a config, a real
// migrated SQLite-backed backend, and an authenticated admin user, ready to
// execute SettingsCommand() against. Returns the config too, so tests can
// mutate AnonAccess/AllowKeyless to simulate a config override.
func newSettingsTestContext(t *testing.T) (context.Context, *config.Config) {
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
	ctx = backend.WithContext(ctx, be)

	// "admin" is already taken by the default user the migration seeds, so
	// use a distinct username for the test's admin.
	admin, err := be.CreateUser(ctx, "testadmin", proto.UserOptions{Admin: true})
	is.NoErr(err)
	ctx = proto.WithUserContext(ctx, admin)

	return ctx, cfg
}

func runSettings(t *testing.T, ctx context.Context, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	c := SettingsCommand()
	var outBuf, errBuf bytes.Buffer
	c.SetOut(&outBuf)
	c.SetErr(&errBuf)
	c.SetArgs(args)
	err = c.ExecuteContext(ctx)
	return outBuf.String(), errBuf.String(), err
}

func TestSettingsAnonAccessWarnsOnConfigOverride(t *testing.T) {
	is := is.New(t)
	ctx, cfg := newSettingsTestContext(t)

	// No override active: setting anon-access should succeed silently.
	_, stderr, err := runSettings(t, ctx, "anon-access", "read-write")
	is.NoErr(err)
	is.Equal(stderr, "")

	// With a config override active, the write still succeeds (so it takes
	// effect if the override is later removed), but must warn loudly that
	// it currently has no effect.
	adminAccess := access.AdminAccess
	cfg.AnonAccess = &adminAccess
	_, stderr, err = runSettings(t, ctx, "anon-access", "no-access")
	is.NoErr(err)
	if !strings.Contains(stderr, "override") {
		t.Fatalf("expected override warning on stderr, got: %q", stderr)
	}
}

func TestSettingsAllowKeylessWarnsOnConfigOverride(t *testing.T) {
	is := is.New(t)
	ctx, cfg := newSettingsTestContext(t)

	// No override active: setting allow-keyless should succeed silently.
	_, stderr, err := runSettings(t, ctx, "allow-keyless", "false")
	is.NoErr(err)
	is.Equal(stderr, "")

	// With a config override active, the write still succeeds, but must
	// warn that it currently has no effect. This is the dangerous
	// direction: an admin trying to lock things down (false) while a
	// config override forces it open (true).
	allow := true
	cfg.AllowKeyless = &allow
	_, stderr, err = runSettings(t, ctx, "allow-keyless", "false")
	is.NoErr(err)
	if !strings.Contains(stderr, "override") {
		t.Fatalf("expected override warning on stderr, got: %q", stderr)
	}
}
