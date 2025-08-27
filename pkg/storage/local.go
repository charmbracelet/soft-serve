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
	name = l.fixPath(name)
	if err := os.Remove(name); err != nil {
		return fmt.Errorf("failed to remove file %s: %w", name, err)
	}
	return nil
}

// Open implements Storage.
func (l *LocalStorage) Open(name string) (Object, error) {
	name = l.fixPath(name)
	f, err := os.Open(name)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", name, err)
	}
	return f, nil
}

// Stat implements Storage.
func (l *LocalStorage) Stat(name string) (fs.FileInfo, error) {
	name = l.fixPath(name)
	info, err := os.Stat(name)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file %s: %w", name, err)
	}
	return info, nil
}

// Put implements Storage.
func (l *LocalStorage) Put(name string, r io.Reader) (int64, error) {
	name = l.fixPath(name)
	if err := os.MkdirAll(filepath.Dir(name), os.ModePerm); err != nil {
		return 0, fmt.Errorf("failed to create directory for %s: %w", name, err)
	}

	f, err := os.Create(name)
	if err != nil {
		return 0, fmt.Errorf("failed to create file %s: %w", name, err)
	}
	defer f.Close()
	n, err := io.Copy(f, r)
	if err != nil {
		return n, fmt.Errorf("failed to copy data to file %s: %w", name, err)
	}
	return n, nil
}

// Exists implements Storage.
func (l *LocalStorage) Exists(name string) (bool, error) {
	name = l.fixPath(name)
	_, err := os.Stat(name)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, fmt.Errorf("failed to check existence of file %s: %w", name, err)
}

// Rename implements Storage.
func (l *LocalStorage) Rename(oldName, newName string) error {
	oldName = l.fixPath(oldName)
	newName = l.fixPath(newName)
	if err := os.MkdirAll(filepath.Dir(newName), os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", newName, err)
	}

	if err := os.Rename(oldName, newName); err != nil {
		return fmt.Errorf("failed to rename %s to %s: %w", oldName, newName, err)
	}
	return nil
}

// Replace all slashes with the OS-specific separator.
func (l LocalStorage) fixPath(path string) string {
	path = strings.ReplaceAll(path, "/", string(os.PathSeparator))
	if !filepath.IsAbs(path) {
		return filepath.Join(l.root, path)
	}

	return path
}
