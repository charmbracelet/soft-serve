package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestRepo creates a temp git repo containing dot_config/bat and returns
// the opened repository and its HEAD reference.
func setupTestRepo(t *testing.T) (*Repository, *Reference) {
	t.Helper()
	ctx := context.Background()

	repoPath := filepath.Join(t.TempDir(), "test-repo")
	nestedDir := filepath.Join(repoPath, "dot_config")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nestedDir, "bat"), []byte("test content"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, args := range [][]string{
		{"init"},
		{"add", "."},
		{"-c", "user.email=test@example.com", "-c", "user.name=Test", "commit", "-m", "init"},
	} {
		cmd := exec.CommandContext(ctx, "git", args...)
		cmd.Dir = repoPath
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s failed: %v\n%s", args[0], err, out)
		}
	}

	repo, err := Open(repoPath)
	if err != nil {
		t.Fatalf("failed to open repository: %v", err)
	}
	ref, err := repo.HEAD()
	if err != nil {
		t.Fatalf("failed to get HEAD: %v", err)
	}
	return repo, ref
}

func TestTreePathForwardSlashes(t *testing.T) {
	repo, ref := setupTestRepo(t)

	// TreePath should clean the double slash and resolve the directory.
	tree, err := repo.TreePath(ref, "dot_config//")
	if err != nil {
		t.Fatalf("TreePath failed: %v", err)
	}

	entries, err := tree.Entries()
	if err != nil {
		t.Fatalf("failed to get entries: %v", err)
	}

	for _, e := range entries {
		path := e.File().Path()
		if strings.Contains(path, `\`) {
			t.Errorf("entry path contains backslash: %q", path)
		}
	}
}

func TestTreeEntryPathForwardSlashes(t *testing.T) {
	repo, ref := setupTestRepo(t)

	tree, err := repo.TreePath(ref, "dot_config")
	if err != nil {
		t.Fatalf("TreePath failed: %v", err)
	}

	entries, err := tree.Entries()
	if err != nil {
		t.Fatalf("failed to get entries: %v", err)
	}

	for _, e := range entries {
		path := e.File().Path()
		if strings.Contains(path, `\`) {
			t.Errorf("entry path contains backslash: %q", path)
		}
		if !strings.Contains(path, "/") {
			t.Errorf("expected forward slash in path: %q", path)
		}
	}
}
