package git

import (
	"errors"

	"github.com/gogs/git-module"
)

var (
	// ErrFileNotFound is returned when a file is not found.
	ErrFileNotFound = errors.New("file not found")
	// ErrDirectoryNotFound is returned when a directory is not found.
	ErrDirectoryNotFound = errors.New("directory not found")
	// ErrReferenceNotExist is returned when a reference does not exist.
	ErrReferenceNotExist = git.ErrReferenceNotExist
	// ErrRevisionNotExist is returned when a revision is not found.
	ErrRevisionNotExist = git.ErrRevisionNotExist
	// ErrNotAGitRepository is returned when the given path is not a Git repository.
	ErrNotAGitRepository = errors.New("not a git repository")
)
