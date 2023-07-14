package proto

import (
	"errors"
)

var (
	// ErrUnauthorized is returned when the user is not authorized to perform action.
	ErrUnauthorized = errors.New("Unauthorized")
	// ErrFileNotFound is returned when the file is not found.
	ErrFileNotFound = errors.New("File not found")
	// ErrRepoNotExist is returned when a repository does not exist.
	ErrRepoNotExist = errors.New("repository does not exist")
	// ErrRepoExist is returned when a repository already exists.
	ErrRepoExist = errors.New("repository already exists")
)
