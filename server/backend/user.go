package backend

import (
	"context"
	"errors"
	"strings"

	"github.com/charmbracelet/soft-serve/server/access"
	"github.com/charmbracelet/soft-serve/server/db"
	"github.com/charmbracelet/soft-serve/server/db/models"
	"github.com/charmbracelet/soft-serve/server/proto"
	"github.com/charmbracelet/soft-serve/server/sshutils"
	"github.com/charmbracelet/soft-serve/server/utils"
	"golang.org/x/crypto/ssh"
)

// AccessLevel returns the access level of a user for a repository.
//
// It implements backend.Backend.
func (d *Backend) AccessLevel(ctx context.Context, repo string, username string) access.AccessLevel {
	user, _ := d.User(ctx, username)
	return d.AccessLevelForUser(ctx, repo, user)
}

// AccessLevelByPublicKey returns the access level of a user's public key for a repository.
//
// It implements backend.Backend.
func (d *Backend) AccessLevelByPublicKey(ctx context.Context, repo string, pk ssh.PublicKey) access.AccessLevel {
	for _, k := range d.cfg.AdminKeys() {
		if sshutils.KeysEqual(pk, k) {
			return access.AdminAccess
		}
	}

	user, _ := d.UserByPublicKey(ctx, pk)
	if user != nil {
		return d.AccessLevel(ctx, repo, user.Username())
	}

	return d.AccessLevel(ctx, repo, "")
}

// AccessLevelForUser returns the access level of a user for a repository.
// TODO: user repository ownership
func (d *Backend) AccessLevelForUser(ctx context.Context, repo string, user proto.User) access.AccessLevel {
	var username string
	anon := d.AnonAccess(ctx)
	if user != nil {
		username = user.Username()
	}

	// If the user is an admin, they have admin access.
	if user != nil && user.IsAdmin() {
		return access.AdminAccess
	}

	// If the repository exists, check if the user is a collaborator.
	r := proto.RepositoryFromContext(ctx)
	if r == nil {
		r, _ = d.Repository(ctx, repo)
	}

	if r != nil {
		// If the user is a collaborator, they have read/write access.
		isCollab, _ := d.IsCollaborator(ctx, repo, username)
		if isCollab {
			if anon > access.ReadWriteAccess {
				return anon
			}
			return access.ReadWriteAccess
		}

		// If the repository is private, the user has no access.
		if r.IsPrivate() {
			return access.NoAccess
		}

		// Otherwise, the user has read-only access.
		return access.ReadOnlyAccess
	}

	if user != nil {
		// If the repository doesn't exist, the user has read/write access.
		if anon > access.ReadWriteAccess {
			return anon
		}

		return access.ReadWriteAccess
	}

	// If the user doesn't exist, give them the anonymous access level.
	return anon
}

// User finds a user by username.
//
// It implements backend.Backend.
func (d *Backend) User(ctx context.Context, username string) (proto.User, error) {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return nil, err
	}

	var m models.User
	var pks []ssh.PublicKey
	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		m, err = d.store.FindUserByUsername(ctx, tx, username)
		if err != nil {
			return err
		}

		pks, err = d.store.ListPublicKeysByUserID(ctx, tx, m.ID)
		return err
	}); err != nil {
		err = db.WrapError(err)
		if errors.Is(err, db.ErrRecordNotFound) {
			return nil, proto.ErrUserNotFound
		}
		return nil, err
	}

	return &user{
		user:       m,
		publicKeys: pks,
	}, nil
}

// UserByPublicKey finds a user by public key.
//
// It implements backend.Backend.
func (d *Backend) UserByPublicKey(ctx context.Context, pk ssh.PublicKey) (proto.User, error) {
	var m models.User
	var pks []ssh.PublicKey
	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		m, err = d.store.FindUserByPublicKey(ctx, tx, pk)
		if err != nil {
			return db.WrapError(err)
		}

		pks, err = d.store.ListPublicKeysByUserID(ctx, tx, m.ID)
		return err
	}); err != nil {
		err = db.WrapError(err)
		if errors.Is(err, db.ErrRecordNotFound) {
			return nil, proto.ErrUserNotFound
		}
		return nil, err
	}

	return &user{
		user:       m,
		publicKeys: pks,
	}, nil
}

// Users returns all users.
//
// It implements backend.Backend.
func (d *Backend) Users(ctx context.Context) ([]string, error) {
	var users []string
	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		ms, err := d.store.GetAllUsers(ctx, tx)
		if err != nil {
			return err
		}

		for _, m := range ms {
			users = append(users, m.Username)
		}

		return nil
	}); err != nil {
		return nil, db.WrapError(err)
	}

	return users, nil
}

// AddPublicKey adds a public key to a user.
//
// It implements backend.Backend.
func (d *Backend) AddPublicKey(ctx context.Context, username string, pk ssh.PublicKey) error {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return err
	}

	return db.WrapError(
		d.db.TransactionContext(ctx, func(tx *db.Tx) error {
			return d.store.AddPublicKeyByUsername(ctx, tx, username, pk)
		}),
	)
}

// CreateUser creates a new user.
//
// It implements backend.Backend.
func (d *Backend) CreateUser(ctx context.Context, username string, opts proto.UserOptions) (proto.User, error) {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return nil, err
	}

	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return d.store.CreateUser(ctx, tx, username, opts.Admin, opts.PublicKeys)
	}); err != nil {
		return nil, db.WrapError(err)
	}

	return d.User(ctx, username)
}

// DeleteUser deletes a user.
//
// It implements backend.Backend.
func (d *Backend) DeleteUser(ctx context.Context, username string) error {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return err
	}

	return db.WrapError(
		d.db.TransactionContext(ctx, func(tx *db.Tx) error {
			return d.store.DeleteUserByUsername(ctx, tx, username)
		}),
	)
}

// RemovePublicKey removes a public key from a user.
//
// It implements backend.Backend.
func (d *Backend) RemovePublicKey(ctx context.Context, username string, pk ssh.PublicKey) error {
	return db.WrapError(
		d.db.TransactionContext(ctx, func(tx *db.Tx) error {
			return d.store.RemovePublicKeyByUsername(ctx, tx, username, pk)
		}),
	)
}

// ListPublicKeys lists the public keys of a user.
func (d *Backend) ListPublicKeys(ctx context.Context, username string) ([]ssh.PublicKey, error) {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return nil, err
	}

	var keys []ssh.PublicKey
	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		keys, err = d.store.ListPublicKeysByUsername(ctx, tx, username)
		return err
	}); err != nil {
		return nil, db.WrapError(err)
	}

	return keys, nil
}

// SetUsername sets the username of a user.
//
// It implements backend.Backend.
func (d *Backend) SetUsername(ctx context.Context, username string, newUsername string) error {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return err
	}

	return db.WrapError(
		d.db.TransactionContext(ctx, func(tx *db.Tx) error {
			return d.store.SetUsernameByUsername(ctx, tx, username, newUsername)
		}),
	)
}

// SetAdmin sets the admin flag of a user.
//
// It implements backend.Backend.
func (d *Backend) SetAdmin(ctx context.Context, username string, admin bool) error {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return err
	}

	return db.WrapError(
		d.db.TransactionContext(ctx, func(tx *db.Tx) error {
			return d.store.SetAdminByUsername(ctx, tx, username, admin)
		}),
	)
}

// SetPassword sets the password of a user.
func (d *Backend) SetPassword(ctx context.Context, username string, rawPassword string) error {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return err
	}

	password, err := HashPassword(rawPassword)
	if err != nil {
		return err
	}

	return db.WrapError(
		d.db.TransactionContext(ctx, func(tx *db.Tx) error {
			return d.store.SetUserPasswordByUsername(ctx, tx, username, password)
		}),
	)
}

type user struct {
	user       models.User
	publicKeys []ssh.PublicKey
}

var _ proto.User = (*user)(nil)

// IsAdmin implements proto.User
func (u *user) IsAdmin() bool {
	return u.user.Admin
}

// PublicKeys implements proto.User
func (u *user) PublicKeys() []ssh.PublicKey {
	return u.publicKeys
}

// Username implements proto.User
func (u *user) Username() string {
	return u.user.Username
}

// ID implements proto.User.
func (u *user) ID() int64 {
	return u.user.ID
}
