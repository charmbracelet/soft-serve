package noop

import (
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/server/backend"
)

var _ backend.Repository = (*repo)(nil)

type repo struct {
	path string
}

// Description implements backend.Repository
func (*repo) Description() string {
	return ""
}

// IsPrivate implements backend.Repository
func (*repo) IsPrivate() bool {
	return false
}

// Name implements backend.Repository
func (*repo) Name() string {
	return ""
}

// Repository implements backend.Repository
func (r *repo) Repository() (*git.Repository, error) {
	return git.Open(r.path)
}
