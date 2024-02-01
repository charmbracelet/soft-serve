package backend

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/sshutils"
	"github.com/charmbracelet/soft-serve/pkg/utils"
	"golang.org/x/crypto/ssh"
)

// User finds a user by username.
//
// It implements backend.Backend.
func (d *Backend) User(ctx context.Context, username string) (proto.User, error) {
	username = strings.ToLower(username)
	if err := utils.ValidateHandle(username); err != nil {
		return nil, err
	}

	var m models.User
	var pks []ssh.PublicKey
	var hl models.Handle
	var ems []proto.UserEmail
	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		m, err = d.store.FindUserByUsername(ctx, tx, username)
		if err != nil {
			return err
		}

		pks, err = d.store.ListPublicKeysByUserID(ctx, tx, m.ID)
		if err != nil {
			return err
		}

		emails, err := d.store.ListUserEmails(ctx, tx, m.ID)
		if err != nil {
			return err
		}

		for _, e := range emails {
			ems = append(ems, &userEmail{e})
		}

		hl, err = d.store.GetHandleByUserID(ctx, tx, m.ID)
		return err
	}); err != nil {
		err = db.WrapError(err)
		if errors.Is(err, db.ErrRecordNotFound) {
			return nil, proto.ErrUserNotFound
		}
		d.logger.Error("error finding user", "username", username, "error", err)
		return nil, err
	}

	return &user{
		user:       m,
		publicKeys: pks,
		handle:     hl,
		emails:     ems,
	}, nil
}

// UserByID finds a user by ID.
func (d *Backend) UserByID(ctx context.Context, id int64) (proto.User, error) {
	var m models.User
	var pks []ssh.PublicKey
	var hl models.Handle
	var ems []proto.UserEmail
	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		m, err = d.store.GetUserByID(ctx, tx, id)
		if err != nil {
			return err
		}

		pks, err = d.store.ListPublicKeysByUserID(ctx, tx, m.ID)
		if err != nil {
			return err
		}

		emails, err := d.store.ListUserEmails(ctx, tx, m.ID)
		if err != nil {
			return err
		}

		for _, e := range emails {
			ems = append(ems, &userEmail{e})
		}

		hl, err = d.store.GetHandleByUserID(ctx, tx, m.ID)
		return err
	}); err != nil {
		err = db.WrapError(err)
		if errors.Is(err, db.ErrRecordNotFound) {
			return nil, proto.ErrUserNotFound
		}
		d.logger.Error("error finding user", "id", id, "error", err)
		return nil, err
	}

	return &user{
		user:       m,
		publicKeys: pks,
		handle:     hl,
		emails:     ems,
	}, nil
}

// UserByPublicKey finds a user by public key.
//
// It implements backend.Backend.
func (d *Backend) UserByPublicKey(ctx context.Context, pk ssh.PublicKey) (proto.User, error) {
	var m models.User
	var pks []ssh.PublicKey
	var hl models.Handle
	var ems []proto.UserEmail
	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		m, err = d.store.FindUserByPublicKey(ctx, tx, pk)
		if err != nil {
			return db.WrapError(err)
		}

		pks, err = d.store.ListPublicKeysByUserID(ctx, tx, m.ID)
		if err != nil {
			return err
		}

		emails, err := d.store.ListUserEmails(ctx, tx, m.ID)
		if err != nil {
			return err
		}

		for _, e := range emails {
			ems = append(ems, &userEmail{e})
		}

		hl, err = d.store.GetHandleByUserID(ctx, tx, m.ID)
		return err
	}); err != nil {
		err = db.WrapError(err)
		if errors.Is(err, db.ErrRecordNotFound) {
			return nil, proto.ErrUserNotFound
		}
		d.logger.Error("error finding user", "pk", sshutils.MarshalAuthorizedKey(pk), "error", err)
		return nil, err
	}

	return &user{
		user:       m,
		publicKeys: pks,
		handle:     hl,
		emails:     ems,
	}, nil
}

// UserByAccessToken finds a user by access token.
// This also validates the token for expiration and returns proto.ErrTokenExpired.
func (d *Backend) UserByAccessToken(ctx context.Context, token string) (proto.User, error) {
	var m models.User
	var pks []ssh.PublicKey
	var hl models.Handle
	var ems []proto.UserEmail
	token = HashToken(token)

	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		t, err := d.store.GetAccessTokenByToken(ctx, tx, token)
		if err != nil {
			return db.WrapError(err)
		}

		if t.ExpiresAt.Valid && t.ExpiresAt.Time.Before(time.Now()) {
			return proto.ErrTokenExpired
		}

		m, err = d.store.FindUserByAccessToken(ctx, tx, token)
		if err != nil {
			return db.WrapError(err)
		}

		pks, err = d.store.ListPublicKeysByUserID(ctx, tx, m.ID)
		if err != nil {
			return err
		}

		emails, err := d.store.ListUserEmails(ctx, tx, m.ID)
		if err != nil {
			return err
		}

		for _, e := range emails {
			ems = append(ems, &userEmail{e})
		}

		hl, err = d.store.GetHandleByUserID(ctx, tx, m.ID)
		return err
	}); err != nil {
		err = db.WrapError(err)
		if errors.Is(err, db.ErrRecordNotFound) {
			return nil, proto.ErrUserNotFound
		}
		d.logger.Error("failed to find user by access token", "err", err, "token", token)
		return nil, err
	}

	return &user{
		user:       m,
		publicKeys: pks,
		handle:     hl,
		emails:     ems,
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

		ids := make([]int64, len(ms))
		for i, m := range ms {
			ids[i] = m.ID
		}

		handles, err := d.store.ListHandlesForIDs(ctx, tx, ids)
		if err != nil {
			return err
		}

		for _, h := range handles {
			users = append(users, h.Handle)
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
	if err := utils.ValidateHandle(username); err != nil {
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
	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return d.store.CreateUser(ctx, tx, username, opts.Admin, opts.PublicKeys, opts.Emails)
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
	if err := utils.ValidateHandle(username); err != nil {
		return err
	}

	return d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		if err := d.store.DeleteUserByUsername(ctx, tx, username); err != nil {
			return db.WrapError(err)
		}

		return d.DeleteUserRepositories(ctx, username)
	})
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
	if err := utils.ValidateHandle(username); err != nil {
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
	if err := utils.ValidateHandle(username); err != nil {
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
	if err := utils.ValidateHandle(username); err != nil {
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
	if err := utils.ValidateHandle(username); err != nil {
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

// AddUserEmail adds an email to a user.
func (d *Backend) AddUserEmail(ctx context.Context, user proto.User, email string) error {
	return db.WrapError(
		d.db.TransactionContext(ctx, func(tx *db.Tx) error {
			return d.store.AddUserEmail(ctx, tx, user.ID(), email, false)
		}),
	)
}

// ListUserEmails lists the emails of a user.
func (d *Backend) ListUserEmails(ctx context.Context, user proto.User) ([]proto.UserEmail, error) {
	var ems []proto.UserEmail
	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		emails, err := d.store.ListUserEmails(ctx, tx, user.ID())
		if err != nil {
			return err
		}

		for _, e := range emails {
			ems = append(ems, &userEmail{e})
		}

		return nil
	}); err != nil {
		return nil, db.WrapError(err)
	}

	return ems, nil
}

// RemoveUserEmail deletes an email for a user.
// The deleted email must not be the primary email.
func (d *Backend) RemoveUserEmail(ctx context.Context, user proto.User, email string) error {
	return db.WrapError(
		d.db.TransactionContext(ctx, func(tx *db.Tx) error {
			return d.store.RemoveUserEmail(ctx, tx, user.ID(), email)
		}),
	)
}

// SetUserPrimaryEmail sets the primary email of a user.
func (d *Backend) SetUserPrimaryEmail(ctx context.Context, user proto.User, email string) error {
	return db.WrapError(
		d.db.TransactionContext(ctx, func(tx *db.Tx) error {
			return d.store.SetUserPrimaryEmail(ctx, tx, user.ID(), email)
		}),
	)
}

type user struct {
	user       models.User
	publicKeys []ssh.PublicKey
	handle     models.Handle
	emails     []proto.UserEmail
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
	return u.handle.Handle
}

// ID implements proto.User.
func (u *user) ID() int64 {
	return u.user.ID
}

// Password implements proto.User.
func (u *user) Password() string {
	if u.user.Password.Valid {
		return u.user.Password.String
	}

	return ""
}

// Emails implements proto.User.
func (u *user) Emails() []proto.UserEmail {
	return u.emails
}

type userEmail struct {
	email models.UserEmail
}

var _ proto.UserEmail = (*userEmail)(nil)

// Email implements proto.UserEmail.
func (e *userEmail) Email() string {
	return e.email.Email
}

// ID implements proto.UserEmail.
func (e *userEmail) ID() int64 {
	return e.email.ID
}

// IsPrimary implements proto.UserEmail.
func (e *userEmail) IsPrimary() bool {
	return e.email.IsPrimary
}
