package filesqlite

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/server/db/sqlite"
	"github.com/charmbracelet/soft-serve/server/store"
	"github.com/charmbracelet/soft-serve/server/utils"
	"github.com/jmoiron/sqlx"
)

// TODO: use billy.Filesystem instead of filepath

// CreateRepository creates a new repository.
//
// It implements store.Backend.
func (d *SqliteStore) CreateRepository(ctx context.Context, name string, opts store.RepositoryOptions) (store.Repository, error) {
	name = utils.SanitizeRepo(name)
	if err := utils.ValidateRepo(name); err != nil {
		return nil, err
	}

	repo := name + ".git"
	rp := filepath.Join(d.reposPath(), repo)

	if err := sqlite.WrapTx(d.db.DBx(), ctx, func(tx *sqlx.Tx) error {
		if _, err := tx.Exec(`INSERT INTO repo (name, project_name, description, private, mirror, hidden, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP);`,
			name, opts.ProjectName, opts.Description, opts.Private, opts.Mirror, opts.Hidden); err != nil {
			return err
		}

		_, err := git.Init(rp, true)
		if err != nil {
			d.logger.Debug("failed to create repository", "err", err)
			return err
		}

		if err := d.updateGitDaemonExportOk(name, opts.Private); err != nil {
			return err
		}

		return d.updateDescriptionFile(name, opts.Description)
	}); err != nil {
		d.logger.Debug("failed to create repository in database", "err", err)
		return nil, sqlite.WrapDbErr(err)
	}

	r := &Repo{
		name: name,
		path: rp,
		db:   d.db.DBx(),
	}

	// Set cache
	d.cache.Set(ctx, cacheKey(name), r)

	return r, nil
}

// ImportRepository imports a repository from remote.
func (d *SqliteStore) ImportRepository(ctx context.Context, name string, remote string, opts store.RepositoryOptions) (store.Repository, error) {
	cfg := d.cfg
	name = utils.SanitizeRepo(name)
	if err := utils.ValidateRepo(name); err != nil {
		return nil, err
	}

	repo := name + ".git"
	rp := filepath.Join(d.reposPath(), repo)

	if _, err := os.Stat(rp); err == nil || os.IsExist(err) {
		return nil, ErrRepoExist
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
					filepath.Join(cfg.DataPath, "ssh", "known_hosts"),
					cfg.SSH.ClientKeyPath,
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
// It implements store.Backend.
func (d *SqliteStore) DeleteRepository(ctx context.Context, name string) error {
	name = utils.SanitizeRepo(name)
	repo := name + ".git"
	rp := filepath.Join(d.reposPath(), repo)

	return sqlite.WrapTx(d.db.DBx(), ctx, func(tx *sqlx.Tx) error {
		// Delete repo from cache
		defer d.cache.Delete(ctx, cacheKey(name))

		if _, err := tx.Exec("DELETE FROM repo WHERE name = ?;", name); err != nil {
			return err
		}

		return os.RemoveAll(rp)
	})
}

// RenameRepository renames a repository.
//
// It implements store.Backend.
func (d *SqliteStore) RenameRepository(ctx context.Context, oldName string, newName string) error {
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
		return ErrRepoNotExist
	}

	if _, err := os.Stat(np); err == nil {
		return ErrRepoExist
	}

	if err := sqlite.WrapTx(d.db.DBx(), ctx, func(tx *sqlx.Tx) error {
		// Delete cache
		defer d.cache.Delete(ctx, cacheKey(oldName))

		_, err := tx.Exec("UPDATE repo SET name = ?, updated_at = CURRENT_TIMESTAMP WHERE name = ?;", newName, oldName)
		if err != nil {
			return err
		}

		// Make sure the new repository parent directory exists.
		if err := os.MkdirAll(filepath.Dir(np), os.ModePerm); err != nil {
			return err
		}

		if err := os.Rename(op, np); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return sqlite.WrapDbErr(err)
	}

	return nil
}

// CountRepositories returns the total number of repositories.
func (d *SqliteStore) CountRepositories(ctx context.Context) (uint64, error) {
	var count uint64

	if err := sqlite.WrapTx(d.db.DBx(), ctx, func(tx *sqlx.Tx) error {
		return tx.QueryRow("SELECT COUNT(*) FROM repo;").Scan(&count)
	}); err != nil {
		return 0, sqlite.WrapDbErr(err)
	}

	return count, nil
}

// Repositories returns a list of all repositories.
//
// It implements store.Backend.
func (d *SqliteStore) Repositories(ctx context.Context, page int, perPage int) ([]store.Repository, error) {
	repos := make([]store.Repository, 0)

	if err := sqlite.WrapTx(d.db.DBx(), ctx, func(tx *sqlx.Tx) error {
		rows, err := tx.Query("SELECT name FROM repo ORDER BY updated_at DESC LIMIT ? OFFSET ?;",
			perPage, (page-1)*perPage)
		if err != nil {
			return err
		}

		defer rows.Close() // nolint: errcheck
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				return err
			}

			if r, ok := d.cache.Get(ctx, cacheKey(name)); ok && r != nil {
				if r, ok := r.(*Repo); ok {
					repos = append(repos, r)
				}
				continue
			}

			r := &Repo{
				name: name,
				path: filepath.Join(d.reposPath(), name+".git"),
				db:   d.db.DBx(),
			}

			// Cache repositories
			d.cache.Set(ctx, cacheKey(name), r)

			repos = append(repos, r)
		}

		return nil
	}); err != nil {
		return nil, sqlite.WrapDbErr(err)
	}

	return repos, nil
}

// Repository returns a repository by name.
//
// It implements store.Backend.
func (d *SqliteStore) Repository(ctx context.Context, repo string) (store.Repository, error) {
	repo = utils.SanitizeRepo(repo)

	if r, ok := d.cache.Get(ctx, cacheKey(repo)); ok && r != nil {
		if r, ok := r.(*Repo); ok {
			return r, nil
		}
	}

	rp := filepath.Join(d.reposPath(), repo+".git")
	if _, err := os.Stat(rp); err != nil {
		return nil, os.ErrNotExist
	}

	var count int
	if err := sqlite.WrapTx(d.db.DBx(), ctx, func(tx *sqlx.Tx) error {
		return tx.Get(&count, "SELECT COUNT(*) FROM repo WHERE name = ?", repo)
	}); err != nil {
		return nil, sqlite.WrapDbErr(err)
	}

	if count == 0 {
		d.logger.Warn("repository exists but not found in database", "repo", repo)
		return nil, ErrRepoNotExist
	}

	r := &Repo{
		name: repo,
		path: rp,
		db:   d.db.DBx(),
	}

	// Add to cache
	d.cache.Set(ctx, cacheKey(repo), r)

	return r, nil
}

// Description returns the description of a repository.
//
// It implements store.Backend.
func (d *SqliteStore) Description(ctx context.Context, repo string) (string, error) {
	r, err := d.Repository(ctx, repo)
	if err != nil {
		return "", err
	}

	return r.Description(), nil
}

// IsMirror returns true if the repository is a mirror.
//
// It implements store.Backend.
func (d *SqliteStore) IsMirror(ctx context.Context, repo string) (bool, error) {
	r, err := d.Repository(ctx, repo)
	if err != nil {
		return false, err
	}

	return r.IsMirror(), nil
}

// IsPrivate returns true if the repository is private.
//
// It implements store.Backend.
func (d *SqliteStore) IsPrivate(ctx context.Context, repo string) (bool, error) {
	r, err := d.Repository(ctx, repo)
	if err != nil {
		return false, err
	}

	return r.IsPrivate(), nil
}

// IsHidden returns true if the repository is hidden.
//
// It implements store.Backend.
func (d *SqliteStore) IsHidden(ctx context.Context, repo string) (bool, error) {
	r, err := d.Repository(ctx, repo)
	if err != nil {
		return false, err
	}

	return r.IsHidden(), nil
}

// SetHidden sets the hidden flag of a repository.
//
// It implements store.Backend.
func (d *SqliteStore) SetHidden(ctx context.Context, repo string, hidden bool) error {
	repo = utils.SanitizeRepo(repo)

	// Delete cache
	d.cache.Delete(ctx, cacheKey(repo))

	return sqlite.WrapDbErr(sqlite.WrapTx(d.db.DBx(), ctx, func(tx *sqlx.Tx) error {
		var count int
		if err := tx.Get(&count, "SELECT COUNT(*) FROM repo WHERE name = ?", repo); err != nil {
			return err
		}
		if count == 0 {
			return ErrRepoNotExist
		}
		_, err := tx.Exec("UPDATE repo SET hidden = ?, updated_at = CURRENT_TIMESTAMP WHERE name = ?;", hidden, repo)
		return err
	}))
}

// ProjectName returns the project name of a repository.
//
// It implements store.Backend.
func (d *SqliteStore) ProjectName(ctx context.Context, repo string) (string, error) {
	r, err := d.Repository(ctx, repo)
	if err != nil {
		return "", err
	}

	return r.ProjectName(), nil
}

func (d *SqliteStore) updateDescriptionFile(name, desc string) error {
	repo := utils.RepoPath(d.reposPath(), name)
	fp := filepath.Join(repo, "description")
	return os.WriteFile(fp, []byte(desc), 0644)
}

// SetDescription sets the description of a repository.
//
// It implements store.Backend.
func (d *SqliteStore) SetDescription(ctx context.Context, repo string, desc string) error {
	repo = utils.SanitizeRepo(repo)

	// Delete cache
	d.cache.Delete(ctx, cacheKey(repo))

	return sqlite.WrapTx(d.db.DBx(), ctx, func(tx *sqlx.Tx) error {
		var count int
		if err := tx.Get(&count, "SELECT COUNT(*) FROM repo WHERE name = ?", repo); err != nil {
			return err
		}
		if count == 0 {
			return ErrRepoNotExist
		}
		_, err := tx.Exec("UPDATE repo SET description = ?, updated_at = CURRENT_TIMESTAMP WHERE name = ?", desc, repo)
		if err != nil {
			return err
		}

		return d.updateDescriptionFile(repo, desc)
	})
}

func (d *SqliteStore) updateGitDaemonExportOk(name string, isPrivate bool) error {
	repo := utils.RepoPath(d.reposPath(), name)
	fp := filepath.Join(repo, "git-daemon-export-ok")
	if isPrivate {
		return os.Remove(fp)
	}
	return os.WriteFile(fp, []byte{}, 0644) // nolint: gosec
}

// SetPrivate sets the private flag of a repository.
//
// It implements store.Backend.
func (d *SqliteStore) SetPrivate(ctx context.Context, name string, private bool) error {
	name = utils.SanitizeRepo(name)

	// Delete cache
	d.cache.Delete(ctx, cacheKey(name))

	return sqlite.WrapDbErr(
		sqlite.WrapTx(d.db.DBx(), ctx, func(tx *sqlx.Tx) error {
			var count int
			if err := tx.Get(&count, "SELECT COUNT(*) FROM repo WHERE name = ?", name); err != nil {
				return err
			}
			if count == 0 {
				return ErrRepoNotExist
			}

			_, err := tx.Exec("UPDATE repo SET private = ?, updated_at = CURRENT_TIMESTAMP WHERE name = ?", private, name)
			if err != nil {
				return err
			}

			return d.updateGitDaemonExportOk(name, private)
		}),
	)
}

// SetProjectName sets the project name of a repository.
//
// It implements store.Backend.
func (d *SqliteStore) SetProjectName(ctx context.Context, repo string, name string) error {
	repo = utils.SanitizeRepo(repo)

	// Delete cache
	d.cache.Delete(ctx, cacheKey(repo))

	return sqlite.WrapDbErr(
		sqlite.WrapTx(d.db.DBx(), ctx, func(tx *sqlx.Tx) error {
			var count int
			if err := tx.Get(&count, "SELECT COUNT(*) FROM repo WHERE name = ?", repo); err != nil {
				return err
			}
			if count == 0 {
				return ErrRepoNotExist
			}
			_, err := tx.Exec("UPDATE repo SET project_name = ?, updated_at = CURRENT_TIMESTAMP WHERE name = ?", name, repo)
			return err
		}),
	)
}

// Touch updates the last update time of a repository.
func (d *SqliteStore) Touch(ctx context.Context, repo string) error {
	repo = utils.SanitizeRepo(repo)

	// Delete cache
	d.cache.Delete(ctx, cacheKey(repo))

	return sqlite.WrapDbErr(
		sqlite.WrapTx(d.db.DBx(), ctx, func(tx *sqlx.Tx) error {
			_, err := tx.Exec("UPDATE repo SET updated_at = CURRENT_TIMESTAMP WHERE name = ?", repo)
			if err != nil {
				return err
			}

			if err := d.populateLastModified(ctx, repo); err != nil {
				d.logger.Error("error populating last-modified", "repo", repo, "err", err)
				return err
			}

			return nil
		}),
	)
}

func (d *SqliteStore) populateLastModified(ctx context.Context, repo string) error {
	var rr *Repo
	_rr, err := d.Repository(ctx, repo)
	if err != nil {
		return err
	}

	if r, ok := _rr.(*Repo); ok {
		rr = r
	} else {
		return ErrRepoNotExist
	}

	r, err := rr.Open()
	if err != nil {
		return err
	}

	c, err := r.LatestCommitTime()
	if err != nil {
		return err
	}

	return rr.writeLastModified(c)
}
