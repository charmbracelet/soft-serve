package store

import "errors"

var (
	// ErrRepoNotExist is returned when a repository does not exist.
	ErrRepoNotExist = errors.New("repository does not exist")

	// ErrRepoExist is returned when a repository already exists.
	ErrRepoExist = errors.New("repository already exists")
)
