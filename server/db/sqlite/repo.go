package sqlite

import (
	"github.com/charmbracelet/soft-serve/proto"
	"github.com/charmbracelet/soft-serve/server/db/types"
)

// Open opens a repository.
func (d *Sqlite) Open(name string) (proto.RepositoryService, error) {
	r, err := d.GetRepo(name)
	if err != nil {
		return nil, err
	}
	return &repository{
		repo: r,
		db:   d,
	}, nil
}

type repository struct {
	repo *types.Repo
	db   *Sqlite
}

// Name returns the repository's name.
func (r *repository) Name() string {
	return r.repo.Name
}

// ProjectName returns the repository's project name.
func (r *repository) ProjectName() string {
	return r.repo.ProjectName
}

// SetProjectName sets the repository's project name.
func (r *repository) SetProjectName(name string) error {
	return r.db.SetRepoProjectName(r.repo.Name, name)
}

// Description returns the repository's description.
func (r *repository) Description() string {
	return r.repo.Description
}

// SetDescription sets the repository's description.
func (r *repository) SetDescription(desc string) error {
	return r.db.SetRepoDescription(r.repo.Name, desc)
}

// IsPrivate returns whether the repository is private.
func (r *repository) IsPrivate() bool {
	return r.repo.Private
}

// SetPrivate sets whether the repository is private.
func (r *repository) SetPrivate(p bool) error {
	return r.db.SetRepoPrivate(r.repo.Name, p)
}
