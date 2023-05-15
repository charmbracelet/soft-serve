package sqlite

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/hooks"
	"github.com/charmbracelet/soft-serve/server/utils"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite" // sqlite driver
)

// SqliteBackend is a backend that uses a SQLite database as a Soft Serve
// backend.
type SqliteBackend struct { //nolint: revive
	cfg    *config.Config
	ctx    context.Context
	dp     string
	db     *sqlx.DB
	logger *log.Logger

	// Repositories cache
	cache *cache
}

var _ backend.Backend = (*SqliteBackend)(nil)

func (d *SqliteBackend) reposPath() string {
	return filepath.Join(d.dp, "repos")
}

// NewSqliteBackend creates a new SqliteBackend.
func NewSqliteBackend(ctx context.Context) (*SqliteBackend, error) {
	cfg := config.FromContext(ctx)
	dataPath := cfg.DataPath
	if err := os.MkdirAll(dataPath, os.ModePerm); err != nil {
		return nil, err
	}

	db, err := sqlx.Connect("sqlite", filepath.Join(dataPath, "soft-serve.db"+
		"?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"))
	if err != nil {
		return nil, err
	}

	d := &SqliteBackend{
		cfg:    cfg,
		ctx:    ctx,
		dp:     dataPath,
		db:     db,
		logger: log.FromContext(ctx).WithPrefix("sqlite"),
	}

	// Set up LRU cache with size 1000
	d.cache = newCache(d, 1000)

	if err := d.init(); err != nil {
		return nil, err
	}

	if err := d.db.Ping(); err != nil {
		return nil, err
	}

	return d, d.initRepos()
}

// WithContext returns a copy of SqliteBackend with the given context.
func (d SqliteBackend) WithContext(ctx context.Context) backend.Backend {
	d.ctx = ctx
	return &d
}

// AllowKeyless returns whether or not keyless access is allowed.
//
// It implements backend.Backend.
func (d *SqliteBackend) AllowKeyless() bool {
	var allow bool
	if err := wrapTx(d.db, d.ctx, func(tx *sqlx.Tx) error {
		return tx.Get(&allow, "SELECT value FROM settings WHERE key = ?;", "allow_keyless")
	}); err != nil {
		return false
	}

	return allow
}

// AnonAccess returns the level of anonymous access.
//
// It implements backend.Backend.
func (d *SqliteBackend) AnonAccess() backend.AccessLevel {
	var level string
	if err := wrapTx(d.db, d.ctx, func(tx *sqlx.Tx) error {
		return tx.Get(&level, "SELECT value FROM settings WHERE key = ?;", "anon_access")
	}); err != nil {
		return backend.NoAccess
	}

	return backend.ParseAccessLevel(level)
}

// SetAllowKeyless sets whether or not keyless access is allowed.
//
// It implements backend.Backend.
func (d *SqliteBackend) SetAllowKeyless(allow bool) error {
	return wrapDbErr(
		wrapTx(d.db, d.ctx, func(tx *sqlx.Tx) error {
			_, err := tx.Exec("UPDATE settings SET value = ?, updated_at = CURRENT_TIMESTAMP WHERE key = ?;", allow, "allow_keyless")
			return err
		}),
	)
}

// SetAnonAccess sets the level of anonymous access.
//
// It implements backend.Backend.
func (d *SqliteBackend) SetAnonAccess(level backend.AccessLevel) error {
	return wrapDbErr(
		wrapTx(d.db, d.ctx, func(tx *sqlx.Tx) error {
			_, err := tx.Exec("UPDATE settings SET value = ?, updated_at = CURRENT_TIMESTAMP WHERE key = ?;", level.String(), "anon_access")
			return err
		}),
	)
}

// CreateRepository creates a new repository.
//
// It implements backend.Backend.
func (d *SqliteBackend) CreateRepository(name string, opts backend.RepositoryOptions) (backend.Repository, error) {
	name = utils.SanitizeRepo(name)
	if err := utils.ValidateRepo(name); err != nil {
		return nil, err
	}

	repo := name + ".git"
	rp := filepath.Join(d.reposPath(), repo)

	if err := wrapTx(d.db, d.ctx, func(tx *sqlx.Tx) error {
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

		return nil
	}); err != nil {
		d.logger.Debug("failed to create repository in database", "err", err)
		return nil, wrapDbErr(err)
	}

	r := &Repo{
		name: name,
		path: rp,
		db:   d.db,
	}

	// Set cache
	d.cache.Set(name, r)

	return r, d.initRepo(name)
}

// ImportRepository imports a repository from remote.
func (d *SqliteBackend) ImportRepository(name string, remote string, opts backend.RepositoryOptions) (backend.Repository, error) {
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
			Context: d.ctx,
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

	return d.CreateRepository(name, opts)
}

// DeleteRepository deletes a repository.
//
// It implements backend.Backend.
func (d *SqliteBackend) DeleteRepository(name string) error {
	name = utils.SanitizeRepo(name)
	repo := name + ".git"
	rp := filepath.Join(d.reposPath(), repo)

	return wrapTx(d.db, d.ctx, func(tx *sqlx.Tx) error {
		// Delete repo from cache
		defer d.cache.Delete(name)

		if _, err := tx.Exec("DELETE FROM repo WHERE name = ?;", name); err != nil {
			return err
		}

		return os.RemoveAll(rp)
	})
}

// RenameRepository renames a repository.
//
// It implements backend.Backend.
func (d *SqliteBackend) RenameRepository(oldName string, newName string) error {
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

	if err := wrapTx(d.db, d.ctx, func(tx *sqlx.Tx) error {
		// Delete cache
		defer d.cache.Delete(oldName)

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
		return wrapDbErr(err)
	}

	return nil
}

// Repositories returns a list of all repositories.
//
// It implements backend.Backend.
func (d *SqliteBackend) Repositories() ([]backend.Repository, error) {
	repos := make([]backend.Repository, 0)

	if err := wrapTx(d.db, d.ctx, func(tx *sqlx.Tx) error {
		rows, err := tx.Query("SELECT name FROM repo")
		if err != nil {
			return err
		}

		defer rows.Close() // nolint: errcheck
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				return err
			}

			if r, ok := d.cache.Get(name); ok && r != nil {
				repos = append(repos, r)
				continue
			}

			r := &Repo{
				name: name,
				path: filepath.Join(d.reposPath(), name+".git"),
				db:   d.db,
			}

			// Cache repositories
			d.cache.Set(name, r)

			repos = append(repos, r)
		}

		return nil
	}); err != nil {
		return nil, wrapDbErr(err)
	}

	return repos, nil
}

// Repository returns a repository by name.
//
// It implements backend.Backend.
func (d *SqliteBackend) Repository(repo string) (backend.Repository, error) {
	repo = utils.SanitizeRepo(repo)

	if r, ok := d.cache.Get(repo); ok && r != nil {
		return r, nil
	}

	rp := filepath.Join(d.reposPath(), repo+".git")
	if _, err := os.Stat(rp); err != nil {
		return nil, os.ErrNotExist
	}

	var count int
	if err := wrapTx(d.db, d.ctx, func(tx *sqlx.Tx) error {
		return tx.Get(&count, "SELECT COUNT(*) FROM repo WHERE name = ?", repo)
	}); err != nil {
		return nil, wrapDbErr(err)
	}

	if count == 0 {
		d.logger.Warn("repository exists but not found in database", "repo", repo)
		return nil, ErrRepoNotExist
	}

	r := &Repo{
		name: repo,
		path: rp,
		db:   d.db,
	}

	// Add to cache
	d.cache.Set(repo, r)

	return r, nil
}

// Description returns the description of a repository.
//
// It implements backend.Backend.
func (d *SqliteBackend) Description(repo string) (string, error) {
	r, err := d.Repository(repo)
	if err != nil {
		return "", err
	}

	return r.Description(), nil
}

// IsMirror returns true if the repository is a mirror.
//
// It implements backend.Backend.
func (d *SqliteBackend) IsMirror(repo string) (bool, error) {
	r, err := d.Repository(repo)
	if err != nil {
		return false, err
	}

	return r.IsMirror(), nil
}

// IsPrivate returns true if the repository is private.
//
// It implements backend.Backend.
func (d *SqliteBackend) IsPrivate(repo string) (bool, error) {
	r, err := d.Repository(repo)
	if err != nil {
		return false, err
	}

	return r.IsPrivate(), nil
}

// IsHidden returns true if the repository is hidden.
//
// It implements backend.Backend.
func (d *SqliteBackend) IsHidden(repo string) (bool, error) {
	r, err := d.Repository(repo)
	if err != nil {
		return false, err
	}

	return r.IsHidden(), nil
}

// SetHidden sets the hidden flag of a repository.
//
// It implements backend.Backend.
func (d *SqliteBackend) SetHidden(repo string, hidden bool) error {
	repo = utils.SanitizeRepo(repo)

	// Delete cache
	d.cache.Delete(repo)

	return wrapDbErr(wrapTx(d.db, d.ctx, func(tx *sqlx.Tx) error {
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
// It implements backend.Backend.
func (d *SqliteBackend) ProjectName(repo string) (string, error) {
	r, err := d.Repository(repo)
	if err != nil {
		return "", err
	}

	return r.ProjectName(), nil
}

// SetDescription sets the description of a repository.
//
// It implements backend.Backend.
func (d *SqliteBackend) SetDescription(repo string, desc string) error {
	repo = utils.SanitizeRepo(repo)

	// Delete cache
	d.cache.Delete(repo)

	return wrapTx(d.db, d.ctx, func(tx *sqlx.Tx) error {
		var count int
		if err := tx.Get(&count, "SELECT COUNT(*) FROM repo WHERE name = ?", repo); err != nil {
			return err
		}
		if count == 0 {
			return ErrRepoNotExist
		}
		_, err := tx.Exec("UPDATE repo SET description = ?, updated_at = CURRENT_TIMESTAMP WHERE name = ?", desc, repo)
		return err
	})
}

// SetPrivate sets the private flag of a repository.
//
// It implements backend.Backend.
func (d *SqliteBackend) SetPrivate(repo string, private bool) error {
	repo = utils.SanitizeRepo(repo)

	// Delete cache
	d.cache.Delete(repo)

	return wrapDbErr(
		wrapTx(d.db, d.ctx, func(tx *sqlx.Tx) error {
			var count int
			if err := tx.Get(&count, "SELECT COUNT(*) FROM repo WHERE name = ?", repo); err != nil {
				return err
			}
			if count == 0 {
				return ErrRepoNotExist
			}
			_, err := tx.Exec("UPDATE repo SET private = ?, updated_at = CURRENT_TIMESTAMP WHERE name = ?", private, repo)
			return err
		}),
	)
}

// SetProjectName sets the project name of a repository.
//
// It implements backend.Backend.
func (d *SqliteBackend) SetProjectName(repo string, name string) error {
	repo = utils.SanitizeRepo(repo)

	// Delete cache
	d.cache.Delete(repo)

	return wrapDbErr(
		wrapTx(d.db, d.ctx, func(tx *sqlx.Tx) error {
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

// AddCollaborator adds a collaborator to a repository.
//
// It implements backend.Backend.
func (d *SqliteBackend) AddCollaborator(repo string, username string) error {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return err
	}

	repo = utils.SanitizeRepo(repo)
	return wrapDbErr(wrapTx(d.db, d.ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`INSERT INTO collab (user_id, repo_id, updated_at)
			VALUES (
			(SELECT id FROM user WHERE username = ?),
			(SELECT id FROM repo WHERE name = ?),
			CURRENT_TIMESTAMP
			);`, username, repo)
		return err
	}),
	)
}

// Collaborators returns a list of collaborators for a repository.
//
// It implements backend.Backend.
func (d *SqliteBackend) Collaborators(repo string) ([]string, error) {
	repo = utils.SanitizeRepo(repo)
	var users []string
	if err := wrapTx(d.db, d.ctx, func(tx *sqlx.Tx) error {
		return tx.Select(&users, `SELECT user.username FROM user
			INNER JOIN collab ON user.id = collab.user_id
			INNER JOIN repo ON repo.id = collab.repo_id
			WHERE repo.name = ?`, repo)
	}); err != nil {
		return nil, wrapDbErr(err)
	}

	return users, nil
}

// IsCollaborator returns true if the user is a collaborator of the repository.
//
// It implements backend.Backend.
func (d *SqliteBackend) IsCollaborator(repo string, username string) (bool, error) {
	repo = utils.SanitizeRepo(repo)
	var count int
	if err := wrapTx(d.db, d.ctx, func(tx *sqlx.Tx) error {
		return tx.Get(&count, `SELECT COUNT(*) FROM user
			INNER JOIN collab ON user.id = collab.user_id
			INNER JOIN repo ON repo.id = collab.repo_id
			WHERE repo.name = ? AND user.username = ?`, repo, username)
	}); err != nil {
		return false, wrapDbErr(err)
	}

	return count > 0, nil
}

// RemoveCollaborator removes a collaborator from a repository.
//
// It implements backend.Backend.
func (d *SqliteBackend) RemoveCollaborator(repo string, username string) error {
	repo = utils.SanitizeRepo(repo)
	return wrapDbErr(
		wrapTx(d.db, d.ctx, func(tx *sqlx.Tx) error {
			_, err := tx.Exec(`DELETE FROM collab
			WHERE user_id = (SELECT id FROM user WHERE username = ?)
			AND repo_id = (SELECT id FROM repo WHERE name = ?)`, username, repo)
			return err
		}),
	)
}

func (d *SqliteBackend) initRepo(repo string) error {
	return hooks.GenerateHooks(d.ctx, d.cfg, repo)
}

func (d *SqliteBackend) initRepos() error {
	repos, err := d.Repositories()
	if err != nil {
		return err
	}

	for _, repo := range repos {
		if err := d.initRepo(repo.Name()); err != nil {
			return err
		}
	}

	return nil
}

// TODO: implement a caching interface.
type cache struct {
	b     *SqliteBackend
	repos *lru.Cache[string, *Repo]
}

func newCache(b *SqliteBackend, size int) *cache {
	if size <= 0 {
		size = 1
	}
	c := &cache{b: b}
	cache, _ := lru.New[string, *Repo](size)
	c.repos = cache
	return c
}

func (c *cache) Get(repo string) (*Repo, bool) {
	return c.repos.Get(repo)
}

func (c *cache) Set(repo string, r *Repo) {
	c.repos.Add(repo, r)
}

func (c *cache) Delete(repo string) {
	c.repos.Remove(repo)
}

func (c *cache) Len() int {
	return c.repos.Len()
}
