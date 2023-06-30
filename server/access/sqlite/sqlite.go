package sqlite

import (
	"context"
	"strings"

	"github.com/charmbracelet/soft-serve/server/access"
	"github.com/charmbracelet/soft-serve/server/auth"
	"github.com/charmbracelet/soft-serve/server/db"
	"github.com/charmbracelet/soft-serve/server/db/sqlite"
	"github.com/charmbracelet/soft-serve/server/settings"
	"github.com/charmbracelet/soft-serve/server/store"
	"github.com/charmbracelet/soft-serve/server/utils"
	"github.com/jmoiron/sqlx"
)

// SqliteAccess is an access backend implementation that uses SQLite.
type SqliteAccess struct {
	ctx context.Context
	db  db.Database
}

var _ access.Access = (*SqliteAccess)(nil)

func init() {
	access.Register("sqlite", newSqliteAccess)
}

func newSqliteAccess(ctx context.Context) (access.Access, error) {
	sdb := db.FromContext(ctx)
	if sdb == nil {
		return nil, db.ErrNoDatabase
	}

	return &SqliteAccess{
		ctx: ctx,
		db:  sdb,
	}, nil
}

// AccessLevel implements access.Access.
func (d *SqliteAccess) AccessLevel(ctx context.Context, repo string, user auth.User) (access.AccessLevel, error) {
	settings := settings.FromContext(d.ctx)
	store := store.FromContext(d.ctx)

	anon := settings.AnonAccess(ctx)

	// TODO: add admin access to user repositories

	// If the user is an admin, they have admin access.
	if user != nil && user.IsAdmin() {
		return access.AdminAccess, nil
	}

	// If the repository exists, check if the user is a collaborator.
	r, _ := store.Repository(ctx, repo)
	if r != nil {
		// If the user is a collaborator, they have read/write access.
		if user != nil {
			isCollab, _ := d.IsCollaborator(ctx, repo, user.Username())
			if isCollab {
				if anon > access.ReadWriteAccess {
					return anon, nil
				}
				return access.ReadWriteAccess, nil
			}
		}

		// If the repository is private, the user has no access.
		if r.IsPrivate() {
			return access.NoAccess, nil
		}

		// Otherwise, the user has read-only access.
		return access.ReadOnlyAccess, nil
	}

	if user != nil {
		// If the repository doesn't exist, the user has read/write access.
		if anon > access.ReadWriteAccess {
			return anon, nil
		}

		return access.ReadWriteAccess, nil
	}

	// If the user doesn't exist, give them the anonymous access level.
	return anon, nil
}

// AddCollaborator adds a collaborator to a repository.
//
// It implements backend.Backend.
func (d *SqliteAccess) AddCollaborator(ctx context.Context, repo string, username string) error {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return err
	}

	repo = utils.SanitizeRepo(repo)
	return sqlite.WrapDbErr(sqlite.WrapTx(d.db.DBx(), ctx, func(tx *sqlx.Tx) error {
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
func (d *SqliteAccess) Collaborators(ctx context.Context, repo string) ([]string, error) {
	repo = utils.SanitizeRepo(repo)
	var users []string
	if err := sqlite.WrapTx(d.db.DBx(), d.ctx, func(tx *sqlx.Tx) error {
		return tx.Select(&users, `SELECT user.username FROM user
			INNER JOIN collab ON user.id = collab.user_id
			INNER JOIN repo ON repo.id = collab.repo_id
			WHERE repo.name = ?`, repo)
	}); err != nil {
		return nil, sqlite.WrapDbErr(err)
	}

	return users, nil
}

// IsCollaborator returns true if the user is a collaborator of the repository.
//
// It implements backend.Backend.
func (d *SqliteAccess) IsCollaborator(ctx context.Context, repo string, username string) (bool, error) {
	repo = utils.SanitizeRepo(repo)
	var count int
	if err := sqlite.WrapTx(d.db.DBx(), ctx, func(tx *sqlx.Tx) error {
		return tx.Get(&count, `SELECT COUNT(*) FROM user
			INNER JOIN collab ON user.id = collab.user_id
			INNER JOIN repo ON repo.id = collab.repo_id
			WHERE repo.name = ? AND user.username = ?`, repo, username)
	}); err != nil {
		return false, sqlite.WrapDbErr(err)
	}

	return count > 0, nil
}

// RemoveCollaborator removes a collaborator from a repository.
//
// It implements backend.Backend.
func (d *SqliteAccess) RemoveCollaborator(ctx context.Context, repo string, username string) error {
	repo = utils.SanitizeRepo(repo)
	return sqlite.WrapDbErr(
		sqlite.WrapTx(d.db.DBx(), ctx, func(tx *sqlx.Tx) error {
			_, err := tx.Exec(`DELETE FROM collab
			WHERE user_id = (SELECT id FROM user WHERE username = ?)
			AND repo_id = (SELECT id FROM repo WHERE name = ?)`, username, repo)
			return err
		}),
	)
}
