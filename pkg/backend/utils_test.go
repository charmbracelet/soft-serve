package backend

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/soft-serve/git"
)

// stubRepo implements proto.Repository backed by a real on-disk git repo.
type stubRepo struct {
	path string
}

func (s *stubRepo) ID() int64            { return 0 }
func (s *stubRepo) Name() string         { return "stub" }
func (s *stubRepo) ProjectName() string  { return "stub" }
func (s *stubRepo) Description() string  { return "" }
func (s *stubRepo) IsPrivate() bool      { return false }
func (s *stubRepo) IsMirror() bool       { return false }
func (s *stubRepo) IsHidden() bool       { return false }
func (s *stubRepo) UserID() int64        { return 0 }
func (s *stubRepo) CreatedAt() time.Time { return time.Time{} }
func (s *stubRepo) UpdatedAt() time.Time { return time.Time{} }
func (s *stubRepo) Open() (*git.Repository, error) {
	return git.Open(s.path)
}

// initTestRepo creates a non-bare git repo in a temp dir, commits the given
// files (map of relative path → content), and returns the path.
func initTestRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()

	repo, err := git.Init(dir, false)
	if err != nil {
		t.Fatalf("git init: %v", err)
	}
	_ = repo

	for rel, content := range files {
		full := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write file %s: %v", rel, err)
		}
	}

	// git add -A && git commit
	cmd := func(args ...string) {
		t.Helper()
		c := git.NewCommand(args...)
		if _, err := c.RunInDir(dir); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}
	cmd("add", "-A")
	cmd("-c", "user.email=test@test.com", "-c", "user.name=Test", "commit", "-m", "init")

	return dir
}

func TestReadme_Root(t *testing.T) {
	dir := initTestRepo(t, map[string]string{
		"README.md": "# Root Readme",
	})
	repo := &stubRepo{path: dir}
	content, path, err := Readme(repo, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "README.md" {
		t.Errorf("expected path README.md, got %q", path)
	}
	if content != "# Root Readme" {
		t.Errorf("unexpected content: %q", content)
	}
}

func TestReadme_DocsSubdir(t *testing.T) {
	// No root README; only docs/README.md
	dir := initTestRepo(t, map[string]string{
		"main.go":        "package main",
		"docs/README.md": "# Docs Readme",
	})
	repo := &stubRepo{path: dir}
	content, path, err := Readme(repo, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "docs/README.md" {
		t.Errorf("expected path docs/README.md, got %q", path)
	}
	if content != "# Docs Readme" {
		t.Errorf("unexpected content: %q", content)
	}
}

func TestReadme_GithubSubdir(t *testing.T) {
	// No root README; only .github/README.md
	dir := initTestRepo(t, map[string]string{
		"main.go":           "package main",
		".github/README.md": "# GitHub Readme",
	})
	repo := &stubRepo{path: dir}
	content, path, err := Readme(repo, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != ".github/README.md" {
		t.Errorf("expected path .github/README.md, got %q", path)
	}
	if content != "# GitHub Readme" {
		t.Errorf("unexpected content: %q", content)
	}
}

func TestReadme_GitlabSubdir(t *testing.T) {
	// No root README; only .gitlab/README.md
	dir := initTestRepo(t, map[string]string{
		"main.go":           "package main",
		".gitlab/README.md": "# GitLab Readme",
	})
	repo := &stubRepo{path: dir}
	content, path, err := Readme(repo, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != ".gitlab/README.md" {
		t.Errorf("expected path .gitlab/README.md, got %q", path)
	}
	if content != "# GitLab Readme" {
		t.Errorf("unexpected content: %q", content)
	}
}

func TestReadme_RootTakesPrecedence(t *testing.T) {
	// Both root and docs have a README; root should win
	dir := initTestRepo(t, map[string]string{
		"README.md":      "# Root Readme",
		"docs/README.md": "# Docs Readme",
	})
	repo := &stubRepo{path: dir}
	_, path, err := Readme(repo, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "README.md" {
		t.Errorf("expected root README.md to take precedence, got %q", path)
	}
}

func TestReadme_NotFound(t *testing.T) {
	dir := initTestRepo(t, map[string]string{
		"main.go": "package main",
	})
	repo := &stubRepo{path: dir}
	content, path, err := Readme(repo, nil)
	if err != nil {
		t.Fatalf("unexpected error when no readme exists: %v", err)
	}
	if path != "" || content != "" {
		t.Errorf("expected empty path and content when no readme exists, got content=%q path=%q", content, path)
	}
}

// initBareTestRepo creates a bare git repo in a temp dir by initialising a
// non-bare repo, committing the given files, and then cloning it as bare.
func initBareTestRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	src := initTestRepo(t, files)
	bareDir := t.TempDir()
	c := git.NewCommand("clone", "--bare", src, bareDir)
	if _, err := c.Run(); err != nil {
		t.Fatalf("git clone --bare: %v", err)
	}
	return bareDir
}

func TestReadme_BareRepo_Root(t *testing.T) {
	dir := initBareTestRepo(t, map[string]string{
		"README.md": "# Bare Readme",
	})
	repo := &stubRepo{path: dir}
	content, path, err := Readme(repo, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "README.md" {
		t.Errorf("expected path README.md, got %q", path)
	}
	if content != "# Bare Readme" {
		t.Errorf("unexpected content: %q", content)
	}
}

func TestReadme_BareRepo_NotFound(t *testing.T) {
	dir := initBareTestRepo(t, map[string]string{
		"main.go": "package main",
	})
	repo := &stubRepo{path: dir}
	content, path, err := Readme(repo, nil)
	if err != nil {
		t.Fatalf("unexpected error when no readme exists in bare repo: %v", err)
	}
	if path != "" || content != "" {
		t.Errorf("expected empty result for bare repo with no readme, got content=%q path=%q", content, path)
	}
}
