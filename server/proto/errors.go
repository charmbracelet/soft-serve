package proto

import (
	"errors"
)

var (
	// ErrUnauthorized is returned when the user is not authorized to perform action.
	ErrUnauthorized = errors.New("unauthorized")
	// ErrFileNotFound is returned when the file is not found.
	ErrFileNotFound = errors.New("file not found")
	// ErrRepoNotFound is returned when a repository does not exist.
	ErrRepoNotFound = errors.New("repository not found")
	// ErrRepoExist is returned when a repository already exists.
	ErrRepoExist = errors.New("repository already exists")
	// ErrUserNotFound is returned when a user does not exist.
	ErrUserNotFound = errors.New("user does not exist")
)
