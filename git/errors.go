package git

import "errors"

var (
	// ErrFileNotFound is returned when a file is not found.
	ErrFileNotFound = errors.New("file not found")
	// ErrDirectoryNotFound is returned when a directory is not found.
	ErrDirectoryNotFound = errors.New("directory not found")
	// ErrReferenceNotFound is returned when a reference is not found.
	ErrReferenceNotFound = errors.New("reference not found")
)
