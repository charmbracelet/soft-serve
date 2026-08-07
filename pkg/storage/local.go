package storage

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ErrPathTraversal is returned when a path attempts to escape the storage root.
var ErrPathTraversal = errors.New("path traversal detected")

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
	name, err := l.fixPath(name)
	if err != nil {
		return err
	}
	return os.Remove(name)
}

// Open implements Storage.
func (l *LocalStorage) Open(name string) (Object, error) {
	name, err := l.fixPath(name)
	if err != nil {
		return nil, err
	}
	return os.Open(name)
}

// Stat implements Storage.
func (l *LocalStorage) Stat(name string) (fs.FileInfo, error) {
	name, err := l.fixPath(name)
	if err != nil {
		return nil, err
	}
	return os.Stat(name)
}

// Put implements Storage.
func (l *LocalStorage) Put(name string, r io.Reader) (int64, error) {
	name, err := l.fixPath(name)
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(filepath.Dir(name), os.ModePerm); err != nil {
		return 0, err
	}

	f, err := os.Create(name)
	if err != nil {
		return 0, err
	}
	defer f.Close() //nolint: errcheck
	return io.Copy(f, r)
}

// Exists implements Storage.
func (l *LocalStorage) Exists(name string) (bool, error) {
	name, err := l.fixPath(name)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(name)
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
	oldName, err := l.fixPath(oldName)
	if err != nil {
		return err
	}
	newName, err = l.fixPath(newName)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(newName), os.ModePerm); err != nil {
		return err
	}

	return os.Rename(oldName, newName)
}

// fixPath resolves the given path relative to the storage root and ensures
// it does not escape outside the root directory.
func (l LocalStorage) fixPath(name string) (string, error) {
	name = strings.ReplaceAll(name, "/", string(os.PathSeparator))
	if filepath.IsAbs(name) {
		return "", ErrPathTraversal
	}

	resolved := filepath.Join(l.root, name)
	// Clean the path to resolve any ".." components.
	resolved = filepath.Clean(resolved)

	// Ensure the resolved path is still under root.
	root := filepath.Clean(l.root)
	if !strings.HasPrefix(resolved, root+string(os.PathSeparator)) && resolved != root {
		return "", ErrPathTraversal
	}

	return resolved, nil
}
