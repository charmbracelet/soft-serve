package storage

import (
	"io"
	"io/fs"
)

// Object is an interface for objects that can be stored.
type Object interface {
	io.Seeker
	fs.File
	Name() string
}

// Storage is an interface for storing and retrieving objects.
type Storage interface {
	Open(name string) (Object, error)
	Stat(name string) (fs.FileInfo, error)
	Put(name string, r io.Reader) (int64, error)
	Delete(name string) error
	Exists(name string) (bool, error)
	Rename(oldName, newName string) error
}
