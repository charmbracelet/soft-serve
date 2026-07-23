package hook

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRunCustomHookPropagatesExitError(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(executable)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&0o111 == 0 {
		t.Skip("test binary is not executable on this platform")
	}

	t.Setenv("SOFT_SERVE_TEST_HELPER", "1")
	t.Setenv("SOFT_SERVE_TEST_HELPER_EXIT", "1")
	err = runCustomHook(
		context.Background(),
		executable,
		io.Reader(nil),
		io.Discard,
		io.Discard,
		"-test.run=TestCustomHookHelperProcess",
	)
	if err == nil {
		t.Fatal("runCustomHook returned nil for a failing custom hook")
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("runCustomHook error = %T %v, want exec.ExitError", err, err)
	}
}

func TestRunCustomHookIgnoresMissingOrNonExecutableHook(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing-hook")
	if err := runCustomHook(context.Background(), missing, nil, io.Discard, io.Discard); err != nil {
		t.Fatalf("runCustomHook(missing) error = %v", err)
	}

	nonExecutable := filepath.Join(t.TempDir(), "non-executable-hook")
	if err := os.WriteFile(nonExecutable, []byte("exit 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runCustomHook(context.Background(), nonExecutable, nil, io.Discard, io.Discard); err != nil {
		t.Fatalf("runCustomHook(non-executable) error = %v", err)
	}
}

func TestCustomHookHelperProcess(t *testing.T) {
	if os.Getenv("SOFT_SERVE_TEST_HELPER") != "1" {
		return
	}
	if os.Getenv("SOFT_SERVE_TEST_HELPER_EXIT") == "1" {
		os.Exit(1)
	}
	os.Exit(0)
}
