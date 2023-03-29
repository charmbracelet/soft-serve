package file

import (
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
	root string
	path string
}

// Name returns the repository's name.
//
// It implements backend.Repository.
func (r *Repo) Name() string {
	name := strings.TrimSuffix(strings.TrimPrefix(r.path, r.root), ".git")
	return strings.TrimPrefix(name, "/")
}

// ProjectName returns the repository's project name.
func (r *Repo) ProjectName() string {
	pn, err := readOneLine(filepath.Join(r.path, projectName))
	if err != nil {
		return ""
	}

	return strings.TrimSpace(pn)
}

// Description returns the repository's description.
//
// It implements backend.Repository.
func (r *Repo) Description() string {
	desc, err := readAll(filepath.Join(r.path, description))
	if err != nil {
		return ""
	}

	return strings.TrimSpace(desc)
}

// IsPrivate returns whether the repository is private.
//
// It implements backend.Repository.
func (r *Repo) IsPrivate() bool {
	_, err := os.Stat(filepath.Join(r.path, private))
	return err == nil
}

// Open returns the underlying git.Repository.
//
// It implements backend.Repository.
func (r *Repo) Open() (*git.Repository, error) {
	return git.Open(r.path)
}
