package sqlite

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/db"
)

var _ backend.Repository = (*Repo)(nil)

// Repo is a Git repository with metadata stored in a SQLite database.
type Repo struct {
	name string
	path string
	db   *db.DB

	// cache
	// updatedAt is cached in "last-modified" file.
	mu          sync.Mutex
	desc        *string
	projectName *string
	isMirror    *bool
	isPrivate   *bool
	isHidden    *bool
}

// Description returns the repository's description.
//
// It implements backend.Repository.
func (r *Repo) Description() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.desc != nil {
		return *r.desc
	}

	var desc string
	if err := r.db.TransactionContext(context.Background(), func(tx *db.Tx) error {
		return tx.Get(&desc, "SELECT description FROM repo WHERE name = ?", r.name)
	}); err != nil {
		return ""
	}

	r.desc = &desc
	return desc
}

// IsMirror returns whether the repository is a mirror.
//
// It implements backend.Repository.
func (r *Repo) IsMirror() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.isMirror != nil {
		return *r.isMirror
	}

	var mirror bool
	if err := r.db.TransactionContext(context.Background(), func(tx *db.Tx) error {
		return tx.Get(&mirror, "SELECT mirror FROM repo WHERE name = ?", r.name)
	}); err != nil {
		return false
	}

	r.isMirror = &mirror
	return mirror
}

// IsPrivate returns whether the repository is private.
//
// It implements backend.Repository.
func (r *Repo) IsPrivate() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.isPrivate != nil {
		return *r.isPrivate
	}

	var private bool
	if err := r.db.TransactionContext(context.Background(), func(tx *db.Tx) error {
		return tx.Get(&private, "SELECT private FROM repo WHERE name = ?", r.name)
	}); err != nil {
		return false
	}

	r.isPrivate = &private
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
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.projectName != nil {
		return *r.projectName
	}

	var name string
	if err := r.db.TransactionContext(context.Background(), func(tx *db.Tx) error {
		return tx.Get(&name, "SELECT project_name FROM repo WHERE name = ?", r.name)
	}); err != nil {
		return ""
	}

	r.projectName = &name
	return name
}

// IsHidden returns whether the repository is hidden.
//
// It implements backend.Repository.
func (r *Repo) IsHidden() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.isHidden != nil {
		return *r.isHidden
	}

	var hidden bool
	if err := r.db.TransactionContext(context.Background(), func(tx *db.Tx) error {
		return tx.Get(&hidden, "SELECT hidden FROM repo WHERE name = ?", r.name)
	}); err != nil {
		return false
	}

	r.isHidden = &hidden
	return hidden
}

// UpdatedAt returns the repository's last update time.
func (r *Repo) UpdatedAt() time.Time {
	var updatedAt time.Time

	// Try to read the last modified time from the info directory.
	if t, err := readOneline(filepath.Join(r.path, "info", "last-modified")); err == nil {
		if t, err := time.Parse(time.RFC3339, t); err == nil {
			return t
		}
	}

	rr, err := git.Open(r.path)
	if err == nil {
		t, err := rr.LatestCommitTime()
		if err == nil {
			updatedAt = t
		}
	}

	if updatedAt.IsZero() {
		if err := r.db.TransactionContext(context.Background(), func(tx *db.Tx) error {
			return tx.Get(&updatedAt, "SELECT updated_at FROM repo WHERE name = ?", r.name)
		}); err != nil {
			return time.Time{}
		}
	}

	return updatedAt
}

func (r *Repo) writeLastModified(t time.Time) error {
	fp := filepath.Join(r.path, "info", "last-modified")
	if err := os.MkdirAll(filepath.Dir(fp), os.ModePerm); err != nil {
		return err
	}

	return os.WriteFile(fp, []byte(t.Format(time.RFC3339)), os.ModePerm)
}

func readOneline(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}

	defer f.Close() // nolint: errcheck
	s := bufio.NewScanner(f)
	s.Scan()
	return s.Text(), s.Err()
}
