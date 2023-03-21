package file

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/server/backend"
)

var _ backend.Repository = (*Repo)(nil)

// Repo is a filesystem Git repository.
//
// It implemenets backend.Repository.
type Repo struct {
	path string
}

// Name returns the repository's name.
//
// It implements backend.Repository.
func (r *Repo) Name() string {
	return strings.TrimSuffix(filepath.Base(r.path), ".git")
}

// Description returns the repository's description.
//
// It implements backend.Repository.
func (r *Repo) Description() string {
	desc, err := readAll(r.path)
	if err != nil {
		logger.Debug("failed to read description file", "err", err,
			"path", filepath.Join(r.path, description))
		return ""
	}

	return desc
}

// IsPrivate returns whether the repository is private.
//
// It implements backend.Repository.
func (r *Repo) IsPrivate() bool {
	_, err := os.Stat(filepath.Join(r.path, private))
	return errors.Is(err, os.ErrExist)
}

// Repository returns the underlying git.Repository.
//
// It implements backend.Repository.
func (r *Repo) Repository() (*git.Repository, error) {
	return git.Open(r.path)
}
