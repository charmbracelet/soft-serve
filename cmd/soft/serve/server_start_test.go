package serve

import (
	"net"
	"testing"
	"time"

	"charm.land/log/v2"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/matryer/is"
)

// freePort reserves a TCP port on localhost and returns its address.
func freePort(tb testing.TB) string {
	tb.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		tb.Fatal(err)
	}
	defer l.Close() //nolint: errcheck
	return l.Addr().String()
}

// TestStartSurfacesBindError asserts that when one listener fails to bind,
// Start returns the error instead of blocking forever on the listeners that
// did bind, and that those listeners are released.
func TestStartSurfacesBindError(t *testing.T) {
	is := is.New(t)
	ctx, cfg, be, dbx := setupTestBackendWithDB(t)
	ctx = backend.WithContext(ctx, be)
	ctx = db.WithContext(ctx, dbx)
	ctx = log.WithContext(ctx, log.New(&testWriter{tb: t}))

	// Occupy the SSH port so soft-serve's bind fails, same code path as
	// EACCES on a privileged port, without needing root.
	blocker, err := net.Listen("tcp", "127.0.0.1:0")
	is.NoErr(err)
	defer blocker.Close() //nolint: errcheck

	cfg.SSH.ListenAddr = blocker.Addr().String()
	cfg.HTTP.ListenAddr = freePort(t)
	cfg.Git.ListenAddr = freePort(t)
	cfg.Stats.ListenAddr = freePort(t)
	is.NoErr(cfg.Validate())

	srv, err := NewServer(ctx)
	is.NoErr(err)

	errc := make(chan error, 1)
	go func() { errc <- srv.Start() }()

	select {
	case err := <-errc:
		is.True(err != nil) // Start must report the bind failure
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return after a listener failed to bind")
	}

	// The listeners that did bind must be shut down.
	for _, addr := range []string{cfg.HTTP.ListenAddr, cfg.Git.ListenAddr, cfg.Stats.ListenAddr} {
		l, err := net.Listen("tcp", addr)
		is.NoErr(err) // port should be free again
		_ = l.Close()
	}
}

type testWriter struct{ tb testing.TB }

func (w *testWriter) Write(p []byte) (int, error) {
	w.tb.Log(string(p))
	return len(p), nil
}
