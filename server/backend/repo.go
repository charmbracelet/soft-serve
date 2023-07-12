package backend

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/server/db"
	"github.com/charmbracelet/soft-serve/server/db/models"
	"github.com/charmbracelet/soft-serve/server/hooks"
	"github.com/charmbracelet/soft-serve/server/store"
	"github.com/charmbracelet/soft-serve/server/utils"
)

func (d *Backend) reposPath() string {
	return filepath.Join(d.cfg.DataPath, "repos")
}

// CreateRepository creates a new repository.
//
// It implements backend.Backend.
func (d *Backend) CreateRepository(ctx context.Context, name string, opts store.RepositoryOptions) (store.Repository, error) {
	name = utils.SanitizeRepo(name)
	if err := utils.ValidateRepo(name); err != nil {
		return nil, err
	}

	repo := name + ".git"
	rp := filepath.Join(d.reposPath(), repo)

	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		if err := d.store.CreateRepo(
			ctx,
			tx,
			name,
			opts.ProjectName,
			opts.Description,
			opts.Private,
			opts.Hidden,
			opts.Mirror,
		); err != nil {
			return err
		}

		_, err := git.Init(rp, true)
		if err != nil {
			d.logger.Debug("failed to create repository", "err", err)
			return err
		}

		return hooks.GenerateHooks(ctx, d.cfg, repo)
	}); err != nil {
		d.logger.Debug("failed to create repository in database", "err", err)
		return nil, db.WrapError(err)
	}

	return d.Repository(ctx, name)
}

// ImportRepository imports a repository from remote.
func (d *Backend) ImportRepository(ctx context.Context, name string, remote string, opts store.RepositoryOptions) (store.Repository, error) {
	name = utils.SanitizeRepo(name)
	if err := utils.ValidateRepo(name); err != nil {
		return nil, err
	}

	repo := name + ".git"
	rp := filepath.Join(d.reposPath(), repo)

	if _, err := os.Stat(rp); err == nil || os.IsExist(err) {
		return nil, store.ErrRepoExist
	}

	copts := git.CloneOptions{
		Bare:   true,
		Mirror: opts.Mirror,
		Quiet:  true,
		CommandOptions: git.CommandOptions{
			Timeout: -1,
			Context: ctx,
			Envs: []string{
				fmt.Sprintf(`GIT_SSH_COMMAND=ssh -o UserKnownHostsFile="%s" -o StrictHostKeyChecking=no -i "%s"`,
					filepath.Join(d.cfg.DataPath, "ssh", "known_hosts"),
					d.cfg.SSH.ClientKeyPath,
				),
			},
		},
		// Timeout: time.Hour,
	}

	if err := git.Clone(remote, rp, copts); err != nil {
		d.logger.Error("failed to clone repository", "err", err, "mirror", opts.Mirror, "remote", remote, "path", rp)
		// Cleanup the mess!
		if rerr := os.RemoveAll(rp); rerr != nil {
			err = errors.Join(err, rerr)
		}
		return nil, err
	}

	return d.CreateRepository(ctx, name, opts)
}

// DeleteRepository deletes a repository.
//
// It implements backend.Backend.
func (d *Backend) DeleteRepository(ctx context.Context, name string) error {
	name = utils.SanitizeRepo(name)
	repo := name + ".git"
	rp := filepath.Join(d.reposPath(), repo)

	return d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		// Delete repo from cache
		defer d.cache.Delete(name)

		if err := d.store.DeleteRepoByName(ctx, tx, name); err != nil {
			return err
		}

		return os.RemoveAll(rp)
	})
}

// RenameRepository renames a repository.
//
// It implements backend.Backend.
func (d *Backend) RenameRepository(ctx context.Context, oldName string, newName string) error {
	oldName = utils.SanitizeRepo(oldName)
	if err := utils.ValidateRepo(oldName); err != nil {
		return err
	}

	newName = utils.SanitizeRepo(newName)
	if err := utils.ValidateRepo(newName); err != nil {
		return err
	}
	oldRepo := oldName + ".git"
	newRepo := newName + ".git"
	op := filepath.Join(d.reposPath(), oldRepo)
	np := filepath.Join(d.reposPath(), newRepo)
	if _, err := os.Stat(op); err != nil {
		return store.ErrRepoNotExist
	}

	if _, err := os.Stat(np); err == nil {
		return store.ErrRepoExist
	}

	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		// Delete cache
		defer d.cache.Delete(oldName)

		if err := d.store.SetRepoNameByName(ctx, tx, oldName, newName); err != nil {
			return err
		}

		// Make sure the new repository parent directory exists.
		if err := os.MkdirAll(filepath.Dir(np), os.ModePerm); err != nil {
			return err
		}

		return os.Rename(op, np)
	}); err != nil {
		return db.WrapError(err)
	}

	return nil
}

// Repositories returns a list of repositories per page.
//
// It implements backend.Backend.
func (d *Backend) Repositories(ctx context.Context) ([]store.Repository, error) {
	repos := make([]store.Repository, 0)

	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		ms, err := d.store.GetAllRepos(ctx, tx)
		if err != nil {
			return err
		}

		for _, m := range ms {
			r := &repo{
				name: m.Name,
				path: filepath.Join(d.reposPath(), m.Name+".git"),
				repo: m,
			}

			// Cache repositories
			d.cache.Set(m.Name, r)

			repos = append(repos, r)
		}

		return nil
	}); err != nil {
		return nil, db.WrapError(err)
	}

	return repos, nil
}

// Repository returns a repository by name.
//
// It implements backend.Backend.
func (d *Backend) Repository(ctx context.Context, name string) (store.Repository, error) {
	var m models.Repo
	name = utils.SanitizeRepo(name)

	if r, ok := d.cache.Get(name); ok && r != nil {
		return r, nil
	}

	rp := filepath.Join(d.reposPath(), name+".git")
	if _, err := os.Stat(rp); err != nil {
		return nil, os.ErrNotExist
	}

	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		m, err = d.store.GetRepoByName(ctx, tx, name)
		return err
	}); err != nil {
		return nil, db.WrapError(err)
	}

	r := &repo{
		name: name,
		path: rp,
		repo: m,
	}

	// Add to cache
	d.cache.Set(name, r)

	return r, nil
}

// Description returns the description of a repository.
//
// It implements backend.Backend.
func (d *Backend) Description(ctx context.Context, name string) (string, error) {
	name = utils.SanitizeRepo(name)
	var desc string
	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		desc, err = d.store.GetRepoDescriptionByName(ctx, tx, name)
		return err
	}); err != nil {
		return "", db.WrapError(err)
	}

	return desc, nil
}

// IsMirror returns true if the repository is a mirror.
//
// It implements backend.Backend.
func (d *Backend) IsMirror(ctx context.Context, name string) (bool, error) {
	name = utils.SanitizeRepo(name)
	var mirror bool
	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		mirror, err = d.store.GetRepoIsMirrorByName(ctx, tx, name)
		return err
	}); err != nil {
		return false, db.WrapError(err)
	}
	return mirror, nil
}

// IsPrivate returns true if the repository is private.
//
// It implements backend.Backend.
func (d *Backend) IsPrivate(ctx context.Context, name string) (bool, error) {
	name = utils.SanitizeRepo(name)
	var private bool
	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		private, err = d.store.GetRepoIsPrivateByName(ctx, tx, name)
		return err
	}); err != nil {
		return false, db.WrapError(err)
	}

	return private, nil
}

// IsHidden returns true if the repository is hidden.
//
// It implements backend.Backend.
func (d *Backend) IsHidden(ctx context.Context, name string) (bool, error) {
	name = utils.SanitizeRepo(name)
	var hidden bool
	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		hidden, err = d.store.GetRepoIsHiddenByName(ctx, tx, name)
		return err
	}); err != nil {
		return false, db.WrapError(err)
	}

	return hidden, nil
}

// ProjectName returns the project name of a repository.
//
// It implements backend.Backend.
func (d *Backend) ProjectName(ctx context.Context, name string) (string, error) {
	name = utils.SanitizeRepo(name)
	var pname string
	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		pname, err = d.store.GetRepoProjectNameByName(ctx, tx, name)
		return err
	}); err != nil {
		return "", db.WrapError(err)
	}

	return pname, nil
}

// SetHidden sets the hidden flag of a repository.
//
// It implements backend.Backend.
func (d *Backend) SetHidden(ctx context.Context, name string, hidden bool) error {
	name = utils.SanitizeRepo(name)

	// Delete cache
	d.cache.Delete(name)

	return db.WrapError(d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return d.store.SetRepoIsHiddenByName(ctx, tx, name, hidden)
	}))
}

// SetDescription sets the description of a repository.
//
// It implements backend.Backend.
func (d *Backend) SetDescription(ctx context.Context, repo string, desc string) error {
	repo = utils.SanitizeRepo(repo)

	// Delete cache
	d.cache.Delete(repo)

	return d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return d.store.SetRepoDescriptionByName(ctx, tx, repo, desc)
	})
}

// SetPrivate sets the private flag of a repository.
//
// It implements backend.Backend.
func (d *Backend) SetPrivate(ctx context.Context, repo string, private bool) error {
	repo = utils.SanitizeRepo(repo)

	// Delete cache
	d.cache.Delete(repo)

	return db.WrapError(
		d.db.TransactionContext(ctx, func(tx *db.Tx) error {
			return d.store.SetRepoIsPrivateByName(ctx, tx, repo, private)
		}),
	)
}

// SetProjectName sets the project name of a repository.
//
// It implements backend.Backend.
func (d *Backend) SetProjectName(ctx context.Context, repo string, name string) error {
	repo = utils.SanitizeRepo(repo)

	// Delete cache
	d.cache.Delete(repo)

	return db.WrapError(
		d.db.TransactionContext(ctx, func(tx *db.Tx) error {
			return d.store.SetRepoProjectNameByName(ctx, tx, repo, name)
		}),
	)
}

var _ store.Repository = (*repo)(nil)

// repo is a Git repository with metadata stored in a SQLite database.
type repo struct {
	name string
	path string
	repo models.Repo
}

// Description returns the repository's description.
//
// It implements backend.Repository.
func (r *repo) Description() string {
	return r.repo.Description
}

// IsMirror returns whether the repository is a mirror.
//
// It implements backend.Repository.
func (r *repo) IsMirror() bool {
	return r.repo.Mirror
}

// IsPrivate returns whether the repository is private.
//
// It implements backend.Repository.
func (r *repo) IsPrivate() bool {
	return r.repo.Private
}

// Name returns the repository's name.
//
// It implements backend.Repository.
func (r *repo) Name() string {
	return r.name
}

// Open opens the repository.
//
// It implements backend.Repository.
func (r *repo) Open() (*git.Repository, error) {
	return git.Open(r.path)
}

// ProjectName returns the repository's project name.
//
// It implements backend.Repository.
func (r *repo) ProjectName() string {
	return r.repo.ProjectName
}

// IsHidden returns whether the repository is hidden.
//
// It implements backend.Repository.
func (r *repo) IsHidden() bool {
	return r.repo.Hidden
}

// UpdatedAt returns the repository's last update time.
func (r *repo) UpdatedAt() time.Time {
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
			return t
		}
	}

	return r.repo.UpdatedAt
}

func (r *repo) writeLastModified(t time.Time) error {
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
