package storage

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Names handed to LocalStorage come from LFS object IDs, which are attacker
// controlled. None of them may resolve outside the root.
func TestLocalStorageConfinesToRoot(t *testing.T) {
	outside := t.TempDir()
	root := filepath.Join(outside, "root")
	secret := filepath.Join(outside, "secret")
	if err := os.WriteFile(secret, []byte("host key"), 0o600); err != nil {
		t.Fatal(err)
	}

	l := NewLocalStorage(root)
	for _, name := range []string{
		"../secret",
		"objects/../../secret",
		"../../../../../../../../etc/passwd",
		secret,
		"/etc/passwd",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := l.Open(name); !errors.Is(err, ErrPathTraversal) {
				t.Errorf("Open: got %v, want ErrPathTraversal", err)
			}
			if _, err := l.Stat(name); !errors.Is(err, ErrPathTraversal) {
				t.Errorf("Stat: got %v, want ErrPathTraversal", err)
			}
			if _, err := l.Exists(name); !errors.Is(err, ErrPathTraversal) {
				t.Errorf("Exists: got %v, want ErrPathTraversal", err)
			}
			if _, err := l.Put(name, strings.NewReader("x")); !errors.Is(err, ErrPathTraversal) {
				t.Errorf("Put: got %v, want ErrPathTraversal", err)
			}
			if err := l.Delete(name); !errors.Is(err, ErrPathTraversal) {
				t.Errorf("Delete: got %v, want ErrPathTraversal", err)
			}
			if err := l.Rename("objects/a", name); !errors.Is(err, ErrPathTraversal) {
				t.Errorf("Rename dst: got %v, want ErrPathTraversal", err)
			}
			if err := l.Rename(name, "objects/a"); !errors.Is(err, ErrPathTraversal) {
				t.Errorf("Rename src: got %v, want ErrPathTraversal", err)
			}
		})
	}

	if _, err := os.Stat(root); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("root was created by a rejected write: %v", err)
	}
	if b, err := os.ReadFile(secret); err != nil || string(b) != "host key" {
		t.Errorf("secret was clobbered: %q %v", b, err)
	}
}

// Ordinary relative names still round-trip through the root. This is the shape
// of an LFS upload: stage under "incomplete", then rename into place. Names
// crossing the Storage boundary are relative, so callers must not feed back the
// absolute path an opened Object reports.
func TestLocalStorageRoundTrip(t *testing.T) {
	root := t.TempDir()
	l := NewLocalStorage(root)

	if _, err := l.Put("incomplete/tmp", strings.NewReader("hello")); err != nil {
		t.Fatal(err)
	}
	if err := l.Rename("incomplete/tmp", "objects/ab/cd/abcd"); err != nil {
		t.Fatal(err)
	}

	exists, err := l.Exists("objects/ab/cd/abcd")
	if err != nil || !exists {
		t.Fatalf("Exists: %v %v", exists, err)
	}

	f, err := l.Open("objects/ab/cd/abcd")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close() //nolint: errcheck

	if got := f.Name(); !strings.HasPrefix(got, root) {
		t.Errorf("Name %q is not under root %q", got, root)
	}
}
