package daemon

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/charmbracelet/soft-serve/pkg/config"
)

// TestServeReturnsErrServerClosedOnShutdown asserts that Serve reports
// ErrServerClosed, not the raw accept error, when the daemon is shut down.
// Callers rely on this sentinel to distinguish a clean stop from a failure.
func TestServeReturnsErrServerClosedOnShutdown(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.DataPath = t.TempDir()
	cfg.Git.ListenAddr = "127.0.0.1:0"
	ctx := config.WithContext(context.Background(), cfg)

	for i := 0; i < 50; i++ {
		d, err := NewGitDaemon(ctx)
		if err != nil {
			t.Fatal(err)
		}
		errc := make(chan error, 1)
		go func() { errc <- d.ListenAndServe() }()
		time.Sleep(time.Millisecond)
		if err := d.Shutdown(context.Background()); err != nil {
			t.Fatalf("shutdown: %v", err)
		}
		select {
		case err := <-errc:
			if !errors.Is(err, ErrServerClosed) {
				t.Fatalf("iteration %d: expected ErrServerClosed, got %v", i, err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("Serve did not return after Shutdown")
		}
	}
}
