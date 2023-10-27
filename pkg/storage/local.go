package storage

import (
	"errors"
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
	return os.Remove(name)
}

// Open implements Storage.
func (l *LocalStorage) Open(name string) (Object, error) {
	name = l.fixPath(name)
	return os.Open(name)
}

// Stat implements Storage.
func (l *LocalStorage) Stat(name string) (fs.FileInfo, error) {
	name = l.fixPath(name)
	return os.Stat(name)
}

// Put implements Storage.
func (l *LocalStorage) Put(name string, r io.Reader) (int64, error) {
	name = l.fixPath(name)
	if err := os.MkdirAll(filepath.Dir(name), os.ModePerm); err != nil {
		return 0, err
	}

	f, err := os.Create(name)
	if err != nil {
		return 0, err
	}
	defer f.Close() // nolint: errcheck
	return io.Copy(f, r)
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
	return false, err
}

// Rename implements Storage.
func (l *LocalStorage) Rename(oldName, newName string) error {
	oldName = l.fixPath(oldName)
	newName = l.fixPath(newName)
	if err := os.MkdirAll(filepath.Dir(newName), os.ModePerm); err != nil {
		return err
	}

	return os.Rename(oldName, newName)
}

// Replace all slashes with the OS-specific separator
func (l LocalStorage) fixPath(path string) string {
	path = strings.ReplaceAll(path, "/", string(os.PathSeparator))
	if !filepath.IsAbs(path) {
		return filepath.Join(l.root, path)
	}

	return path
}
