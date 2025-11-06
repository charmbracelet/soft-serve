package backend

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/hooks"
	"github.com/charmbracelet/soft-serve/pkg/lfs"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/storage"
	"github.com/charmbracelet/soft-serve/pkg/task"
	"github.com/charmbracelet/soft-serve/pkg/utils"
	"github.com/charmbracelet/soft-serve/pkg/webhook"
)

// CreateRepository creates a new repository.
//
// It implements backend.Backend.
func (d *Backend) CreateRepository(ctx context.Context, name string, user proto.User, opts proto.RepositoryOptions) (proto.Repository, error) {
	name = utils.SanitizeRepo(name)
	if err := utils.ValidateRepo(name); err != nil {
		return nil, err
	}

	rp := filepath.Join(d.repoPath(name))

	var userID int64
	if user != nil {
		userID = user.ID()
	}

	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		if err := d.store.CreateRepo(
			ctx,
			tx,
			name,
			userID,
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

		if err := os.WriteFile(filepath.Join(rp, "description"), []byte(opts.Description), fs.ModePerm); err != nil {
			d.logger.Error("failed to write description", "repo", name, "err", err)
			return err
		}

		if !opts.Private {
			if err := os.WriteFile(filepath.Join(rp, "git-daemon-export-ok"), []byte{}, fs.ModePerm); err != nil {
				d.logger.Error("failed to write git-daemon-export-ok", "repo", name, "err", err)
				return err
			}
		}

		return hooks.GenerateHooks(ctx, d.cfg, name)
	}); err != nil {
		d.logger.Debug("failed to create repository in database", "err", err)
		err = db.WrapError(err)
		if errors.Is(err, db.ErrDuplicateKey) {
			return nil, proto.ErrRepoExist
		}

		return nil, err
	}

	return d.Repository(ctx, name)
}

// ImportRepository imports a repository from remote.
// XXX: This a expensive operation and should be run in a goroutine.
func (d *Backend) ImportRepository(_ context.Context, name string, user proto.User, remote string, opts proto.RepositoryOptions) (proto.Repository, error) {
	name = utils.SanitizeRepo(name)
	if err := utils.ValidateRepo(name); err != nil {
		return nil, err
	}

	rp := filepath.Join(d.repoPath(name))

	tid := "import:" + name
	if d.manager.Exists(tid) {
		return nil, task.ErrAlreadyStarted
	}

	if _, err := os.Stat(rp); err == nil || os.IsExist(err) {
		return nil, proto.ErrRepoExist
	}

	done := make(chan error, 1)
	repoc := make(chan proto.Repository, 1)
	d.logger.Info("importing repository", "name", name, "remote", remote, "path", rp)
	d.manager.Add(tid, func(ctx context.Context) (err error) {
		ctx = proto.WithUserContext(ctx, user)

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
		}

		if err := git.Clone(remote, rp, copts); err != nil {
			d.logger.Error("failed to clone repository", "err", err, "mirror", opts.Mirror, "remote", remote, "path", rp)
			// Cleanup the mess!
			if rerr := os.RemoveAll(rp); rerr != nil {
				err = errors.Join(err, rerr)
			}

			return err
		}

		r, err := d.CreateRepository(ctx, name, user, opts)
		if err != nil {
			d.logger.Error("failed to create repository", "err", err, "name", name)
			return err
		}

		defer func() {
			if err != nil {
				if rerr := d.DeleteRepository(ctx, name); rerr != nil {
					d.logger.Error("failed to delete repository", "err", rerr, "name", name)
				}
			}
		}()

		rr, err := r.Open()
		if err != nil {
			d.logger.Error("failed to open repository", "err", err, "path", rp)
			return err
		}

		repoc <- r

		rcfg, err := rr.Config()
		if err != nil {
			d.logger.Error("failed to get repository config", "err", err, "path", rp)
			return err
		}

		endpoint := remote
		if opts.LFSEndpoint != "" {
			endpoint = opts.LFSEndpoint
		}

		rcfg.Section("lfs").SetOption("url", endpoint)

		if err := rr.SetConfig(rcfg); err != nil {
			d.logger.Error("failed to set repository config", "err", err, "path", rp)
			return err
		}

		ep, err := lfs.NewEndpoint(endpoint)
		if err != nil {
			d.logger.Error("failed to create lfs endpoint", "err", err, "path", rp)
			return err
		}

		client := lfs.NewClient(ep)
		if client == nil {
			d.logger.Warn("failed to create lfs client: unsupported endpoint", "endpoint", endpoint)
			return nil
		}

		if err := StoreRepoMissingLFSObjects(ctx, r, d.db, d.store, client); err != nil {
			d.logger.Error("failed to store missing lfs objects", "err", err, "path", rp)
			return err
		}

		return nil
	})

	go func() {
		d.logger.Info("running import", "name", name)
		d.manager.Run(tid, done)
	}()

	return <-repoc, <-done
}

// DeleteRepository deletes a repository.
//
// It implements backend.Backend.
func (d *Backend) DeleteRepository(ctx context.Context, name string) error {
	name = utils.SanitizeRepo(name)
	rp := filepath.Join(d.repoPath(name))

	user := proto.UserFromContext(ctx)
	r, err := d.Repository(ctx, name)
	if err != nil {
		return err
	}

	// We create the webhook event before deleting the repository so we can
	// send the event after deleting the repository.
	wh, err := webhook.NewRepositoryEvent(ctx, user, r, webhook.RepositoryEventActionDelete)
	if err != nil {
		return err
	}

	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		// Delete repo from cache
		defer d.cache.Delete(name)

		repom, dberr := d.store.GetRepoByName(ctx, tx, name)
		_, ferr := os.Stat(rp)
		if dberr != nil && ferr != nil {
			return proto.ErrRepoNotFound
		}

		// If the repo is not in the database but the directory exists, remove it
		if dberr != nil && ferr == nil {
			return os.RemoveAll(rp)
		} else if dberr != nil {
			return db.WrapError(dberr)
		}

		repoID := strconv.FormatInt(repom.ID, 10)
		strg := storage.NewLocalStorage(filepath.Join(d.cfg.DataPath, "lfs", repoID))
		objs, err := d.store.GetLFSObjectsByName(ctx, tx, name)
		if err != nil {
			return db.WrapError(err)
		}

		for _, obj := range objs {
			p := lfs.Pointer{
				Oid:  obj.Oid,
				Size: obj.Size,
			}

			d.logger.Debug("deleting lfs object", "repo", name, "oid", obj.Oid)
			if err := strg.Delete(path.Join("objects", p.RelativePath())); err != nil {
				d.logger.Error("failed to delete lfs object", "repo", name, "err", err, "oid", obj.Oid)
			}
		}

		if err := d.store.DeleteRepoByName(ctx, tx, name); err != nil {
			return db.WrapError(err)
		}

		return os.RemoveAll(rp)
	}); err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			return proto.ErrRepoNotFound
		}

		return db.WrapError(err)
	}

	return webhook.SendEvent(ctx, wh)
}

// DeleteUserRepositories deletes all user repositories.
func (d *Backend) DeleteUserRepositories(ctx context.Context, username string) error {
	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		user, err := d.store.FindUserByUsername(ctx, tx, username)
		if err != nil {
			return err
		}

		repos, err := d.store.GetUserRepos(ctx, tx, user.ID)
		if err != nil {
			return err
		}

		for _, repo := range repos {
			if err := d.DeleteRepository(ctx, repo.Name); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return db.WrapError(err)
	}

	return nil
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

	if oldName == newName {
		return nil
	}

	op := filepath.Join(d.repoPath(oldName))
	np := filepath.Join(d.repoPath(newName))
	if _, err := os.Stat(op); err != nil {
		return proto.ErrRepoNotFound
	}

	if _, err := os.Stat(np); err == nil {
		return proto.ErrRepoExist
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

	user := proto.UserFromContext(ctx)
	repo, err := d.Repository(ctx, newName)
	if err != nil {
		return err
	}

	wh, err := webhook.NewRepositoryEvent(ctx, user, repo, webhook.RepositoryEventActionRename)
	if err != nil {
		return err
	}

	return webhook.SendEvent(ctx, wh)
}

// Repositories returns a list of repositories per page.
//
// It implements backend.Backend.
func (d *Backend) Repositories(ctx context.Context) ([]proto.Repository, error) {
	repos := make([]proto.Repository, 0)

	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		ms, err := d.store.GetAllRepos(ctx, tx)
		if err != nil {
			return err
		}

		for _, m := range ms {
			r := &repo{
				name: m.Name,
				path: filepath.Join(d.repoPath(m.Name)),
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
func (d *Backend) Repository(ctx context.Context, name string) (proto.Repository, error) {
	var m models.Repo
	name = utils.SanitizeRepo(name)

	if r, ok := d.cache.Get(name); ok && r != nil {
		return r, nil
	}

	rp := filepath.Join(d.repoPath(name))
	if _, err := os.Stat(rp); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			d.logger.Errorf("failed to stat repository path: %v", err)
		}
		return nil, proto.ErrRepoNotFound
	}

	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		m, err = d.store.GetRepoByName(ctx, tx, name)
		return db.WrapError(err)
	}); err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			return nil, proto.ErrRepoNotFound
		}
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
func (d *Backend) SetDescription(ctx context.Context, name string, desc string) error {
	name = utils.SanitizeRepo(name)
	desc = utils.Sanitize(desc)
	rp := filepath.Join(d.repoPath(name))

	// Delete cache
	d.cache.Delete(name)

	return d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		if err := os.WriteFile(filepath.Join(rp, "description"), []byte(desc), fs.ModePerm); err != nil {
			d.logger.Error("failed to write description", "repo", name, "err", err)
			return err
		}

		return d.store.SetRepoDescriptionByName(ctx, tx, name, desc)
	})
}

// SetPrivate sets the private flag of a repository.
//
// It implements backend.Backend.
func (d *Backend) SetPrivate(ctx context.Context, name string, private bool) error {
	name = utils.SanitizeRepo(name)
	rp := filepath.Join(d.repoPath(name))

	// Delete cache
	d.cache.Delete(name)

	if err := db.WrapError(
		d.db.TransactionContext(ctx, func(tx *db.Tx) error {
			fp := filepath.Join(rp, "git-daemon-export-ok")
			if !private {
				if err := os.WriteFile(fp, []byte{}, fs.ModePerm); err != nil {
					d.logger.Error("failed to write git-daemon-export-ok", "repo", name, "err", err)
					return err
				}
			} else {
				if _, err := os.Stat(fp); err == nil {
					if err := os.Remove(fp); err != nil {
						d.logger.Error("failed to remove git-daemon-export-ok", "repo", name, "err", err)
						return err
					}
				}
			}

			return d.store.SetRepoIsPrivateByName(ctx, tx, name, private)
		}),
	); err != nil {
		return err
	}

	user := proto.UserFromContext(ctx)
	repo, err := d.Repository(ctx, name)
	if err != nil {
		return err
	}

	if repo.IsPrivate() != !private {
		wh, err := webhook.NewRepositoryEvent(ctx, user, repo, webhook.RepositoryEventActionVisibilityChange)
		if err != nil {
			return err
		}

		if err := webhook.SendEvent(ctx, wh); err != nil {
			return err
		}
	}

	return nil
}

// SetProjectName sets the project name of a repository.
//
// It implements backend.Backend.
func (d *Backend) SetProjectName(ctx context.Context, repo string, name string) error {
	repo = utils.SanitizeRepo(repo)
	name = utils.Sanitize(name)

	// Delete cache
	d.cache.Delete(repo)

	return db.WrapError(
		d.db.TransactionContext(ctx, func(tx *db.Tx) error {
			return d.store.SetRepoProjectNameByName(ctx, tx, repo, name)
		}),
	)
}

// repoPath returns the path to a repository.
func (d *Backend) repoPath(name string) string {
	name = utils.SanitizeRepo(name)
	rn := strings.ReplaceAll(name, "/", string(os.PathSeparator))
	return filepath.Join(filepath.Join(d.cfg.DataPath, "repos"), rn+".git")
}

var _ proto.Repository = (*repo)(nil)

// repo is a Git repository with metadata stored in a SQLite database.
type repo struct {
	name string
	path string
	repo models.Repo
}

// ID returns the repository's ID.
//
// It implements proto.Repository.
func (r *repo) ID() int64 {
	return r.repo.ID
}

// UserID returns the repository's owner's user ID.
// If the repository is not owned by anyone, it returns 0.
//
// It implements proto.Repository.
func (r *repo) UserID() int64 {
	if r.repo.UserID.Valid {
		return r.repo.UserID.Int64
	}
	return 0
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

// CreatedAt returns the repository's creation time.
func (r *repo) CreatedAt() time.Time {
	return r.repo.CreatedAt
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

	return os.WriteFile(fp, []byte(t.Format(time.RFC3339)), os.ModePerm) //nolint:gosec
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
