package cmd

import (
	"bytes"
	"context"
	"testing"

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

// newAuthTestContext returns a context wired with a config and a real
// migrated SQLite-backed backend, plus the backend itself so tests can set up
// users and repositories.
//
// No user is attached to the returned context; use withUser to authenticate
// as somebody.
func newAuthTestContext(t *testing.T) (context.Context, *backend.Backend) {
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
	ctx = db.WithContext(ctx, dbx)
	dbstore := database.New(ctx, dbx)
	ctx = store.WithContext(ctx, dbstore)
	be := backend.New(ctx, cfg, dbx, dbstore)
	ctx = backend.WithContext(ctx, be)

	return ctx, be
}

// withUser creates a user and returns a context authenticated as them.
func withUser(t *testing.T, ctx context.Context, be *backend.Backend, username string, admin bool) context.Context {
	t.Helper()
	is := is.New(t)
	user, err := be.CreateUser(ctx, username, proto.UserOptions{Admin: admin})
	is.NoErr(err)
	return proto.WithUserContext(ctx, user)
}

// runRepo runs a `repo` subcommand and returns only its error.
func runRepo(t *testing.T, ctx context.Context, args ...string) error {
	t.Helper()
	_, _, err := runRepoOutput(t, ctx, args...)
	return err
}

// runRepoOutput runs a `repo` subcommand and returns its stdout and stderr
// alongside the error, so tests can assert that sensitive values were not
// printed even when a command fails.
func runRepoOutput(t *testing.T, ctx context.Context, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	c := RepoCommand()
	var outBuf, errBuf bytes.Buffer
	c.SetOut(&outBuf)
	c.SetErr(&errBuf)
	c.SetArgs(args)
	err = c.ExecuteContext(ctx)
	return outBuf.String(), errBuf.String(), err
}

// runUser runs a `user` subcommand and returns only its error.
func runUser(t *testing.T, ctx context.Context, args ...string) error {
	t.Helper()
	c := UserCommand()
	var outBuf, errBuf bytes.Buffer
	c.SetOut(&outBuf)
	c.SetErr(&errBuf)
	c.SetArgs(args)
	return c.ExecuteContext(ctx)
}
