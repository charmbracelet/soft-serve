package sqlite

import (
	"errors"
	"fmt"
)

var (
	// ErrDuplicateKey is returned when a unique constraint is violated.
	ErrDuplicateKey = errors.New("record already exists")

	// ErrNoRecord is returned when a record is not found.
	ErrNoRecord = errors.New("record not found")

	// ErrRepoNotExist is returned when a repository does not exist.
	ErrRepoNotExist = fmt.Errorf("repository does not exist")

	// ErrRepoExist is returned when a repository already exists.
	ErrRepoExist = fmt.Errorf("repository already exists")
)
