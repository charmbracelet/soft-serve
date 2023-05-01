package sqlite

import (
	"context"
	"strings"

	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/utils"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/ssh"
)

// User represents a user.
type User struct {
	username string
	db       *sqlx.DB
}

var _ backend.User = (*User)(nil)

// IsAdmin returns whether the user is an admin.
//
// It implements backend.User.
func (u *User) IsAdmin() bool {
	var admin bool
	if err := wrapTx(u.db, context.Background(), func(tx *sqlx.Tx) error {
		return tx.Get(&admin, "SELECT admin FROM user WHERE username = ?", u.username)
	}); err != nil {
		return false
	}

	return admin
}

// PublicKeys returns the user's public keys.
//
// It implements backend.User.
func (u *User) PublicKeys() []ssh.PublicKey {
	var keys []ssh.PublicKey
	if err := wrapTx(u.db, context.Background(), func(tx *sqlx.Tx) error {
		var keyStrings []string
		if err := tx.Select(&keyStrings, `SELECT public_key
			FROM public_key
			INNER JOIN user ON user.id = public_key.user_id
			WHERE user.username = ?;`, u.username); err != nil {
			return err
		}

		for _, keyString := range keyStrings {
			key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(keyString))
			if err != nil {
				return err
			}
			keys = append(keys, key)
		}

		return nil
	}); err != nil {
		return nil
	}

	return keys
}

// Username returns the user's username.
//
// It implements backend.User.
func (u *User) Username() string {
	return u.username
}

// AccessLevel returns the access level of a user for a repository.
//
// It implements backend.Backend.
func (d *SqliteBackend) AccessLevel(repo string, username string) backend.AccessLevel {
	anon := d.AnonAccess()
	user, _ := d.User(username)
	// If the user is an admin, they have admin access.
	if user != nil && user.IsAdmin() {
		return backend.AdminAccess
	}

	// If the repository exists, check if the user is a collaborator.
	r, _ := d.Repository(repo)
	if r != nil {
		// If the user is a collaborator, they have read/write access.
		isCollab, _ := d.IsCollaborator(repo, username)
		if isCollab {
			if anon > backend.ReadWriteAccess {
				return anon
			}
			return backend.ReadWriteAccess
		}

		// If the repository is private, the user has no access.
		if r.IsPrivate() {
			return backend.NoAccess
		}

		// Otherwise, the user has read-only access.
		return backend.ReadOnlyAccess
	}

	if user != nil {
		// If the repository doesn't exist, the user has read/write access.
		if anon > backend.ReadWriteAccess {
			return anon
		}

		return backend.ReadWriteAccess
	}

	// If the user doesn't exist, give them the anonymous access level.
	return anon
}

// AccessLevelByPublicKey returns the access level of a user's public key for a repository.
//
// It implements backend.Backend.
func (d *SqliteBackend) AccessLevelByPublicKey(repo string, pk ssh.PublicKey) backend.AccessLevel {
	for _, k := range d.cfg.AdminKeys() {
		if backend.KeysEqual(pk, k) {
			return backend.AdminAccess
		}
	}

	user, _ := d.UserByPublicKey(pk)
	if user != nil {
		return d.AccessLevel(repo, user.Username())
	}

	return d.AccessLevel(repo, "")
}

// AddPublicKey adds a public key to a user.
//
// It implements backend.Backend.
func (d *SqliteBackend) AddPublicKey(username string, pk ssh.PublicKey) error {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return err
	}

	return wrapDbErr(
		wrapTx(d.db, context.Background(), func(tx *sqlx.Tx) error {
			var userID int
			if err := tx.Get(&userID, "SELECT id FROM user WHERE username = ?", username); err != nil {
				return err
			}

			_, err := tx.Exec(`INSERT INTO public_key (user_id, public_key, updated_at)
			VALUES (?, ?, CURRENT_TIMESTAMP);`, userID, backend.MarshalAuthorizedKey(pk))
			return err
		}),
	)
}

// CreateUser creates a new user.
//
// It implements backend.Backend.
func (d *SqliteBackend) CreateUser(username string, opts backend.UserOptions) (backend.User, error) {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return nil, err
	}

	var user *User
	if err := wrapTx(d.db, context.Background(), func(tx *sqlx.Tx) error {
		stmt, err := tx.Prepare("INSERT INTO user (username, admin, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP);")
		if err != nil {
			return err
		}

		defer stmt.Close() // nolint: errcheck
		r, err := stmt.Exec(username, opts.Admin)
		if err != nil {
			return err
		}

		if len(opts.PublicKeys) > 0 {
			userID, err := r.LastInsertId()
			if err != nil {
				d.logger.Error("error getting last insert id")
				return err
			}

			for _, pk := range opts.PublicKeys {
				stmt, err := tx.Prepare(`INSERT INTO public_key (user_id, public_key, updated_at)
					VALUES (?, ?, CURRENT_TIMESTAMP);`)
				if err != nil {
					return err
				}

				defer stmt.Close() // nolint: errcheck
				if _, err := stmt.Exec(userID, backend.MarshalAuthorizedKey(pk)); err != nil {
					return err
				}
			}
		}

		user = &User{
			db:       d.db,
			username: username,
		}
		return nil
	}); err != nil {
		return nil, wrapDbErr(err)
	}

	return user, nil
}

// DeleteUser deletes a user.
//
// It implements backend.Backend.
func (d *SqliteBackend) DeleteUser(username string) error {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return err
	}

	return wrapDbErr(
		wrapTx(d.db, context.Background(), func(tx *sqlx.Tx) error {
			_, err := tx.Exec("DELETE FROM user WHERE username = ?", username)
			return err
		}),
	)
}

// RemovePublicKey removes a public key from a user.
//
// It implements backend.Backend.
func (d *SqliteBackend) RemovePublicKey(username string, pk ssh.PublicKey) error {
	return wrapDbErr(
		wrapTx(d.db, context.Background(), func(tx *sqlx.Tx) error {
			_, err := tx.Exec(`DELETE FROM public_key
			WHERE user_id = (SELECT id FROM user WHERE username = ?)
			AND public_key = ?;`, username, backend.MarshalAuthorizedKey(pk))
			return err
		}),
	)
}

// ListPublicKeys lists the public keys of a user.
func (d *SqliteBackend) ListPublicKeys(username string) ([]ssh.PublicKey, error) {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return nil, err
	}

	keys := make([]ssh.PublicKey, 0)
	if err := wrapTx(d.db, context.Background(), func(tx *sqlx.Tx) error {
		var keyStrings []string
		if err := tx.Select(&keyStrings, `SELECT public_key
			FROM public_key
			INNER JOIN user ON user.id = public_key.user_id
			WHERE user.username = ?;`, username); err != nil {
			return err
		}

		for _, keyString := range keyStrings {
			key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(keyString))
			if err != nil {
				return err
			}
			keys = append(keys, key)
		}

		return nil
	}); err != nil {
		return nil, wrapDbErr(err)
	}

	return keys, nil
}

// SetUsername sets the username of a user.
//
// It implements backend.Backend.
func (d *SqliteBackend) SetUsername(username string, newUsername string) error {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return err
	}

	return wrapDbErr(
		wrapTx(d.db, context.Background(), func(tx *sqlx.Tx) error {
			_, err := tx.Exec("UPDATE user SET username = ? WHERE username = ?", newUsername, username)
			return err
		}),
	)
}

// SetAdmin sets the admin flag of a user.
//
// It implements backend.Backend.
func (d *SqliteBackend) SetAdmin(username string, admin bool) error {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return err
	}

	return wrapDbErr(
		wrapTx(d.db, context.Background(), func(tx *sqlx.Tx) error {
			_, err := tx.Exec("UPDATE user SET admin = ? WHERE username = ?", admin, username)
			return err
		}),
	)
}

// User finds a user by username.
//
// It implements backend.Backend.
func (d *SqliteBackend) User(username string) (backend.User, error) {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return nil, err
	}

	if err := wrapTx(d.db, context.Background(), func(tx *sqlx.Tx) error {
		return tx.Get(&username, "SELECT username FROM user WHERE username = ?", username)
	}); err != nil {
		return nil, wrapDbErr(err)
	}

	return &User{
		db:       d.db,
		username: username,
	}, nil
}

// UserByPublicKey finds a user by public key.
//
// It implements backend.Backend.
func (d *SqliteBackend) UserByPublicKey(pk ssh.PublicKey) (backend.User, error) {
	var username string
	if err := wrapTx(d.db, context.Background(), func(tx *sqlx.Tx) error {
		return tx.Get(&username, `SELECT user.username
			FROM public_key
			INNER JOIN user ON user.id = public_key.user_id
			WHERE public_key.public_key = ?;`, backend.MarshalAuthorizedKey(pk))
	}); err != nil {
		return nil, wrapDbErr(err)
	}

	return &User{
		db:       d.db,
		username: username,
	}, nil
}

// Users returns all users.
//
// It implements backend.Backend.
func (d *SqliteBackend) Users() ([]string, error) {
	var users []string
	if err := wrapTx(d.db, context.Background(), func(tx *sqlx.Tx) error {
		return tx.Select(&users, "SELECT username FROM user")
	}); err != nil {
		return nil, wrapDbErr(err)
	}

	return users, nil
}
