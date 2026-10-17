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
	"github.com/charmbracelet/soft-serve/pkg/ssrf"
	"github.com/charmbracelet/soft-serve/pkg/storage"
	"github.com/charmbracelet/soft-serve/pkg/task"
	"github.com/charmbracelet/soft-serve/pkg/utils"
	"github.com/charmbracelet/soft-serve/pkg/webhook"
)

// validateImportRemote validates an import remote against private, internal,
// and loopback ranges, and returns the git environment that must be applied to
// the clone subprocess for that validation to hold.
func validateImportRemote(remote string) ([]string, error) {
	v, err := ssrf.ValidateGitRemote(remote)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", proto.ErrInvalidRemote, err)
	}

	return ssrf.GitEnv(v), nil
}

// CreateRepository creates a new repository.
//
// It implements backend.Backend.
func (d *Backend) CreateRepository(ctx context.Context, name string, user proto.User, opts proto.RepositoryOptions) (proto.Repository, error) {
	name = utils.SanitizeRepo(name)
	if err := utils.ValidateRepo(name); err != nil {
		return nil, err
	}

	rp := d.repoPath(name)
	if err := claimRepoPath(rp); err != nil {
		return nil, err
	}

	if _, err := git.Init(rp, true); err != nil {
		d.logger.Debug("failed to create repository", "err", err)
		os.RemoveAll(rp) //nolint:errcheck
		return nil, err
	}

	return d.registerRepository(ctx, name, user, opts)
}

// claimRepoPath creates the repository directory, failing if anything already
// occupies it. The filesystem decides what "already" means, which is what
// makes this safe on case- and normalization-insensitive volumes.
func claimRepoPath(rp string) error {
	if err := os.MkdirAll(filepath.Dir(rp), os.ModePerm); err != nil {
		return err
	}
	if err := os.Mkdir(rp, os.ModePerm); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return proto.ErrRepoExist
		}
		return err
	}
	return nil
}

// registerRepository records a bare repository that already exists at the
// name's path and finishes setting it up. The directory is removed on failure.
func (d *Backend) registerRepository(ctx context.Context, name string, user proto.User, opts proto.RepositoryOptions) (proto.Repository, error) {
	rp := d.repoPath(name)

	var userID int64
	if user != nil {
		userID = user.ID()
	}

	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		if userID == 0 {
			// Anonymous callers (e.g. an anon-access/allow-keyless
			// override) have no user of their own, but repos.user_id is
			// NOT NULL. Fall back to the default admin as owner, the same
			// "ownerless repos belong to the first admin" idiom already
			// used when migrating repos from the pre-user schema (see
			// 0001_create_tables.go).
			adminID, err := d.defaultAdminUserID(ctx, tx)
			if err != nil {
				return err
			}
			userID = adminID
		}

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
		os.RemoveAll(rp) //nolint:errcheck
		err = db.WrapError(err)
		if errors.Is(err, db.ErrDuplicateKey) {
			return nil, proto.ErrRepoExist
		}

		return nil, err
	}

	return d.Repository(ctx, name)
}

// defaultAdminUserID returns the ID of the lowest-ID admin user, used to
// own repositories created by callers with no user of their own (anonymous
// access via an anon-access/allow-keyless override). The migration that
// seeds the initial database always creates an admin user, so this should
// never fail to find one in practice.
func (d *Backend) defaultAdminUserID(ctx context.Context, tx *db.Tx) (int64, error) {
	users, err := d.store.GetAllUsers(ctx, tx)
	if err != nil {
		return 0, err
	}

	var adminID int64
	for _, u := range users {
		if u.Admin && (adminID == 0 || u.ID < adminID) {
			adminID = u.ID
		}
	}

	if adminID == 0 {
		return 0, errors.New("no admin user found to own anonymous repository")
	}

	return adminID, nil
}

// ImportRepository imports a repository from remote.
// XXX: This a expensive operation and should be run in a goroutine.
func (d *Backend) ImportRepository(_ context.Context, name string, user proto.User, remote string, opts proto.RepositoryOptions) (proto.Repository, error) {
	name = utils.SanitizeRepo(name)
	if err := utils.ValidateRepo(name); err != nil {
		return nil, err
	}

	remote = utils.Sanitize(remote)
	remoteEnv, err := validateImportRemote(remote)
	if err != nil {
		return nil, err
	}

	// The LFS endpoint is user-supplied and gets persisted to the repo config,
	// where the mirror job later reads it. Validate it up front rather than
	// relying on the LFS client to refuse the dial.
	if opts.LFSEndpoint != "" {
		if _, err := ssrf.ValidateGitRemote(opts.LFSEndpoint); err != nil {
			return nil, fmt.Errorf("%w: %w", proto.ErrInvalidRemote, err)
		}
	}

	// Resolve the LFS endpoint before anything is created, so a remote that
	// has no usable one (scp syntax, IPv6 literals) imports without LFS
	// instead of being cloned and then torn down.
	lfsEndpoint := remote
	if opts.LFSEndpoint != "" {
		lfsEndpoint = opts.LFSEndpoint
	}
	ep, err := lfs.NewEndpoint(lfsEndpoint)
	if err != nil {
		if opts.LFSEndpoint != "" {
			return nil, fmt.Errorf("%w: %w", proto.ErrInvalidRemote, err)
		}
		lfsEndpoint, ep = "", nil
	}

	rp := d.repoPath(name)

	tid := "import:" + name
	if d.manager.Exists(tid) {
		return nil, task.ErrAlreadyStarted
	}

	if err := claimRepoPath(rp); err != nil {
		return nil, err
	}

	done := make(chan error, 1)
	repoc := make(chan proto.Repository, 1)
	d.logger.Info("importing repository", "name", name, "remote", remote, "path", rp)
	d.manager.Add(tid, func(ctx context.Context) (err error) {
		ctx = proto.WithUserContext(ctx, user)

		// remoteEnv carries the SSRF guard: it disables redirect following
		// and pins the validated address. Without it the clone can reach a
		// different host than the one validated above.
		cloneEnv := append(remoteEnv,
			fmt.Sprintf(`GIT_SSH_COMMAND=ssh -o UserKnownHostsFile="%s" -o StrictHostKeyChecking=no -i "%s"`,
				filepath.Join(d.cfg.DataPath, "ssh", "known_hosts"),
				d.cfg.SSH.ClientKeyPath,
			),
		)

		copts := git.CloneOptions{
			Bare:   true,
			Mirror: opts.Mirror,
			Quiet:  true,
			CommandOptions: git.CommandOptions{
				Timeout: -1,
				Context: ctx,
				Envs:    cloneEnv,
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

		r, err := d.registerRepository(ctx, name, user, opts)
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

		if ep == nil {
			return nil
		}

		rcfg, err := rr.Config()
		if err != nil {
			d.logger.Error("failed to get repository config", "err", err, "path", rp)
			return err
		}

		rcfg.Section("lfs").SetOption("url", lfsEndpoint)

		if err := rr.SetConfig(rcfg); err != nil {
			d.logger.Error("failed to set repository config", "err", err, "path", rp)
			return err
		}

		client := lfs.NewClient(ep)
		if client == nil {
			d.logger.Warn("failed to create lfs client: unsupported endpoint", "endpoint", lfsEndpoint)
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
	rp := d.repoPath(name)

	// Build the webhook event before deleting so it can be sent after. A
	// repository whose directory is gone has no event, but its database row
	// still has to go.
	var wh *webhook.RepositoryEvent
	r, err := d.Repository(ctx, name)
	switch {
	case err == nil:
		ev, err := webhook.NewRepositoryEvent(ctx, proto.UserFromContext(ctx), r, webhook.RepositoryEventActionDelete)
		if err != nil {
			return err
		}
		wh = &ev
	case !errors.Is(err, proto.ErrRepoNotFound):
		return err
	}

	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		repom, dberr := d.store.GetRepoByName(ctx, tx, name)
		if dberr != nil {
			// A directory with no row is an orphan; remove it.
			if d.repoPathExists(name) {
				return os.RemoveAll(rp)
			}
			if errors.Is(db.WrapError(dberr), db.ErrRecordNotFound) {
				return proto.ErrRepoNotFound
			}
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
		return wrapRepoError(err)
	}

	if wh == nil {
		return nil
	}

	return webhook.SendEvent(ctx, *wh)
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

	op := d.repoPath(oldName)
	np := d.repoPath(newName)
	if !d.repoPathExists(oldName) {
		return proto.ErrRepoNotFound
	}

	if d.repoPathExists(newName) {
		return proto.ErrRepoExist
	}

	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		if err := d.store.SetRepoNameByName(ctx, tx, oldName, newName); err != nil {
			return err
		}

		// Make sure the new repository parent directory exists.
		if err := os.MkdirAll(filepath.Dir(np), os.ModePerm); err != nil {
			d.logger.Error("failed to create repository parent directory", "repo", newName, "err", err)
			return errRepoUpdate
		}

		if err := os.Rename(op, np); err != nil {
			d.logger.Error("failed to move repository", "repo", oldName, "err", err)
			return errRepoUpdate
		}

		return nil
	}); err != nil {
		return wrapRepoError(err)
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

	rp := d.repoPath(name)
	if _, err := os.Stat(rp); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			d.logger.Errorf("failed to stat repository path: %v", err)
		}
		return nil, proto.ErrRepoNotFound
	}

	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		m, err = d.store.GetRepoByName(ctx, tx, name)
		return err
	}); err != nil {
		return nil, wrapRepoError(err)
	}

	r := &repo{
		name: name,
		path: rp,
		repo: m,
	}

	return r, nil
}

// errRepoUpdate replaces filesystem errors, which carry the server data
// path, when a repository setting cannot be written.
var errRepoUpdate = errors.New("failed to update repository")

// wrapRepoError maps a missing database row to proto.ErrRepoNotFound so
// callers never see the raw driver error for a repository that does not
// exist.
func wrapRepoError(err error) error {
	err = db.WrapError(err)
	if errors.Is(err, db.ErrRecordNotFound) {
		return proto.ErrRepoNotFound
	}
	return err
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
		return "", wrapRepoError(err)
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
		return false, wrapRepoError(err)
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
		return false, wrapRepoError(err)
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
		return false, wrapRepoError(err)
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
		return "", wrapRepoError(err)
	}

	return pname, nil
}

// SetHidden sets the hidden flag of a repository.
//
// It implements backend.Backend.
func (d *Backend) SetHidden(ctx context.Context, name string, hidden bool) error {
	name = utils.SanitizeRepo(name)

	// An UPDATE that matches no row succeeds silently, so look the
	// repository up first.
	return wrapRepoError(d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		if _, err := d.store.GetRepoByName(ctx, tx, name); err != nil {
			return err
		}
		return d.store.SetRepoIsHiddenByName(ctx, tx, name, hidden)
	}))
}

// SetMirror sets the mirror flag of a repository.
// Note: enabling mirror mode requires the repository to have been imported
// with a remote URL. Use ImportRepository to create a new mirror.
func (d *Backend) SetMirror(ctx context.Context, name string, mirror bool) error {
	name = utils.SanitizeRepo(name)
	rp := d.repoPath(name)

	return wrapRepoError(d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		if _, err := d.store.GetRepoByName(ctx, tx, name); err != nil {
			return err
		}

		// Update git config
		r, err := git.Open(rp)
		if err != nil {
			d.logger.Error("failed to open repository", "err", err, "path", rp)
			return errRepoUpdate
		}

		rcfg, err := r.Config()
		if err != nil {
			d.logger.Error("failed to read repository config", "err", err, "path", rp)
			return errRepoUpdate
		}

		// Update mirror option for all remotes
		remoteSection := rcfg.Section("remote")
		hasRemote := false
		for _, sub := range remoteSection.Subsections {
			// Check if this remote has a URL
			for _, opt := range sub.Options {
				if opt.Key == "url" && opt.Value != "" {
					hasRemote = true
					found := false
					for i, opt := range sub.Options {
						if opt.Key == "mirror" {
							found = true
							if mirror {
								sub.Options[i].Value = "true"
							} else {
								sub.Options = append(sub.Options[:i], sub.Options[i+1:]...)
							}
							break
						}
					}
					if !found && mirror {
						sub.SetOption("mirror", "true")
					}
					break
				}
			}
		}

		if mirror && !hasRemote {
			return errors.New("cannot enable mirror mode: repository has no remote URL configured")
		}

		if err := r.SetConfig(rcfg); err != nil {
			d.logger.Error("failed to set repository config", "err", err, "path", rp)
			return errRepoUpdate
		}

		return d.store.SetRepoIsMirrorByName(ctx, tx, name, mirror)
	}))
}

// SetDescription sets the description of a repository.
//
// It implements backend.Backend.
func (d *Backend) SetDescription(ctx context.Context, name string, desc string) error {
	name = utils.SanitizeRepo(name)
	desc = utils.Sanitize(desc)
	rp := d.repoPath(name)

	return wrapRepoError(d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		if _, err := d.store.GetRepoByName(ctx, tx, name); err != nil {
			return err
		}

		// The raw error carries the server data path, so log it and return
		// a generic one.
		if err := os.WriteFile(filepath.Join(rp, "description"), []byte(desc), fs.ModePerm); err != nil {
			d.logger.Error("failed to write description", "repo", name, "err", err)
			return errRepoUpdate
		}

		return d.store.SetRepoDescriptionByName(ctx, tx, name, desc)
	}))
}

// SetPrivate sets the private flag of a repository.
//
// It implements backend.Backend.
func (d *Backend) SetPrivate(ctx context.Context, name string, private bool) error {
	name = utils.SanitizeRepo(name)
	rp := d.repoPath(name)

	var changed bool
	if err := wrapRepoError(
		d.db.TransactionContext(ctx, func(tx *db.Tx) error {
			m, err := d.store.GetRepoByName(ctx, tx, name)
			if err != nil {
				return err
			}
			changed = m.Private != private

			fp := filepath.Join(rp, "git-daemon-export-ok")
			if !private {
				if err := os.WriteFile(fp, []byte{}, fs.ModePerm); err != nil {
					d.logger.Error("failed to write git-daemon-export-ok", "repo", name, "err", err)
					return errRepoUpdate
				}
			} else {
				if _, err := os.Stat(fp); err == nil {
					if err := os.Remove(fp); err != nil {
						d.logger.Error("failed to remove git-daemon-export-ok", "repo", name, "err", err)
						return errRepoUpdate
					}
				}
			}

			return d.store.SetRepoIsPrivateByName(ctx, tx, name, private)
		}),
	); err != nil {
		return err
	}

	if !changed {
		return nil
	}

	repo, err := d.Repository(ctx, name)
	if err != nil {
		return err
	}

	wh, err := webhook.NewRepositoryEvent(ctx, proto.UserFromContext(ctx), repo, webhook.RepositoryEventActionVisibilityChange)
	if err != nil {
		return err
	}

	return webhook.SendEvent(ctx, wh)
}

// SetProjectName sets the project name of a repository.
//
// It implements backend.Backend.
func (d *Backend) SetProjectName(ctx context.Context, repo string, name string) error {
	repo = utils.SanitizeRepo(repo)
	name = utils.Sanitize(name)

	return wrapRepoError(
		d.db.TransactionContext(ctx, func(tx *db.Tx) error {
			if _, err := d.store.GetRepoByName(ctx, tx, repo); err != nil {
				return err
			}
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

// repoPathExists reports whether anything occupies the repository's directory,
// whether or not a database row points at it.
func (d *Backend) repoPathExists(name string) bool {
	_, err := os.Lstat(d.repoPath(name))
	return err == nil
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

	defer f.Close() //nolint: errcheck
	s := bufio.NewScanner(f)
	s.Scan()
	return s.Text(), s.Err()
}
