package sqlite

import (
	"context"

	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/jmoiron/sqlx"
)

var _ backend.Repository = (*Repo)(nil)

// Repo is a Git repository with metadata stored in a SQLite database.
type Repo struct {
	name string
	path string
	db   *sqlx.DB
}

// Description returns the repository's description.
//
// It implements backend.Repository.
func (r *Repo) Description() string {
	var desc string
	if err := wrapTx(r.db, context.Background(), func(tx *sqlx.Tx) error {
		return tx.Get(&desc, "SELECT description FROM repo WHERE name = ?", r.name)
	}); err != nil {
		return ""
	}

	return desc
}

// IsMirror returns whether the repository is a mirror.
//
// It implements backend.Repository.
func (r *Repo) IsMirror() bool {
	var mirror bool
	if err := wrapTx(r.db, context.Background(), func(tx *sqlx.Tx) error {
		return tx.Get(&mirror, "SELECT mirror FROM repo WHERE name = ?", r.name)
	}); err != nil {
		return false
	}

	return mirror
}

// IsPrivate returns whether the repository is private.
//
// It implements backend.Repository.
func (r *Repo) IsPrivate() bool {
	var private bool
	if err := wrapTx(r.db, context.Background(), func(tx *sqlx.Tx) error {
		return tx.Get(&private, "SELECT private FROM repo WHERE name = ?", r.name)
	}); err != nil {
		return false
	}

	return private
}

// Name returns the repository's name.
//
// It implements backend.Repository.
func (r *Repo) Name() string {
	return r.name
}

// Open opens the repository.
//
// It implements backend.Repository.
func (r *Repo) Open() (*git.Repository, error) {
	return git.Open(r.path)
}

// ProjectName returns the repository's project name.
//
// It implements backend.Repository.
func (r *Repo) ProjectName() string {
	var name string
	if err := wrapTx(r.db, context.Background(), func(tx *sqlx.Tx) error {
		return tx.Get(&name, "SELECT project_name FROM repo WHERE name = ?", r.name)
	}); err != nil {
		return ""
	}

	return name
}

// IsHidden returns whether the repository is hidden.
//
// It implements backend.Repository.
func (r *Repo) IsHidden() bool {
	var hidden bool
	if err := wrapTx(r.db, context.Background(), func(tx *sqlx.Tx) error {
		return tx.Get(&hidden, "SELECT hidden FROM repo WHERE name = ?", r.name)
	}); err != nil {
		return false
	}

	return hidden
}
