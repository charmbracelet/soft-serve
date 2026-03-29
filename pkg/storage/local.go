package storage

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// LocalStorage is a storage implementation that stores objects on the local
// filesystem.
type LocalStorage struct {
	root string
}

var _ Storage = (*LocalStorage)(nil)

// NewLocalStorage creates a new LocalStorage.
func NewLocalStorage(root string) *LocalStorage {
	return &LocalStorage{root: root}
}

// Delete implements Storage.
func (l *LocalStorage) Delete(name string) error {
	p, err := l.fixPath(name)
	if err != nil {
		return err
	}
	return os.Remove(p)
}

// Open implements Storage.
func (l *LocalStorage) Open(name string) (Object, error) {
	p, err := l.fixPath(name)
	if err != nil {
		return nil, err
	}
	return os.Open(p)
}

// Stat implements Storage.
func (l *LocalStorage) Stat(name string) (fs.FileInfo, error) {
	p, err := l.fixPath(name)
	if err != nil {
		return nil, err
	}
	return os.Stat(p)
}

// Put implements Storage.
func (l *LocalStorage) Put(name string, r io.Reader) (int64, error) {
	p, err := l.fixPath(name)
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(filepath.Dir(p), os.ModePerm); err != nil {
		return 0, err
	}

	f, err := os.Create(p)
	if err != nil {
		return 0, err
	}
	defer f.Close() //nolint: errcheck
	return io.Copy(f, r)
}

// Exists implements Storage.
func (l *LocalStorage) Exists(name string) (bool, error) {
	p, err := l.fixPath(name)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(p)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, err
}

// Rename implements Storage.
func (l *LocalStorage) Rename(oldName, newName string) error {
	oldPath, err := l.fixPath(oldName)
	if err != nil {
		return err
	}
	newPath, err := l.fixPath(newName)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(newPath), os.ModePerm); err != nil {
		return err
	}

	// If destination already exists the object was uploaded concurrently.
	// Remove the temp file and return success — the stored copy is authoritative.
	if _, err := os.Stat(newPath); err == nil {
		_ = os.Remove(oldPath)
		return nil
	}

	return os.Rename(oldPath, newPath)
}

// fixPath resolves the storage-relative path and verifies it stays within the root.
// Replace all slashes with the OS-specific separator.
func (l LocalStorage) fixPath(path string) (string, error) {
	if l.root == "" {
		return "", fmt.Errorf("storage: empty root path")
	}
	path = strings.ReplaceAll(path, "/", string(os.PathSeparator))
	p := filepath.Join(l.root, path)
	// Ensure the resolved path is within the storage root.
	// Note: filepath.Join (used to build p) already resolves ".." sequences,
	// so path traversal via ".." is not possible. The HasPrefix check guards
	// against any residual escapes.
	root := l.root + string(filepath.Separator)
	if !strings.HasPrefix(p, root) && p != l.root {
		return "", fmt.Errorf("storage: path %q escapes root", path)
	}
	return p, nil
}
