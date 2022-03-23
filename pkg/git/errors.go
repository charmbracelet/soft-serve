package git

import "errors"

var (
	ErrFileNotFound      = errors.New("file not found")
	ErrDirectoryNotFound = errors.New("directory not found")
	ErrReferenceNotFound = errors.New("reference not found")
)
