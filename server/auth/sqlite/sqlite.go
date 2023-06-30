package sqlite

import (
	"context"
	"errors"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/server/auth"
	"github.com/charmbracelet/soft-serve/server/db"
	"github.com/charmbracelet/soft-serve/server/db/sqlite"
	"github.com/charmbracelet/soft-serve/server/sshutils"
	"github.com/charmbracelet/soft-serve/server/utils"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/ssh"
)

// SqliteAuthStore is a sqlite auth store.
type SqliteAuthStore struct {
	db     db.Database
	ctx    context.Context
	logger *log.Logger
}

func init() {
	auth.Register("sqlite", newAuthStore)
}

func newAuthStore(ctx context.Context) (auth.Auth, error) {
	sdb := db.FromContext(ctx)
	if sdb == nil {
		return nil, db.ErrNoDatabase
	}

	if _, ok := sdb.(*sqlite.Sqlite); !ok {
		return nil, errors.New("database is not a SQLite database")
	}

	return &SqliteAuthStore{
		db:     sdb,
		ctx:    ctx,
		logger: log.FromContext(ctx).WithPrefix("sqlite"),
	}, nil
}

// Authenticate implements auth.Auth.
func (d *SqliteAuthStore) Authenticate(ctx context.Context, method auth.AuthMethod) (auth.User, error) {
	switch m := method.(type) {
	case auth.PublicKey:
		u, err := d.UserByPublicKey(ctx, m)
		if err != nil {
			return nil, err
		}

		return u, nil
	default:
		return nil, auth.ErrUnsupportedAuthMethod
	}
}

// CreateUser creates a new user.
func (d *SqliteAuthStore) CreateUser(ctx context.Context, username string, opts UserOptions) (auth.User, error) {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return nil, err
	}

	var user *User
	if err := sqlite.WrapTx(d.db.DBx(), ctx, func(tx *sqlx.Tx) error {
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
				if _, err := stmt.Exec(userID, sshutils.MarshalAuthorizedKey(pk)); err != nil {
					return err
				}
			}
		}

		user = &User{
			username:   username,
			isAdmin:    opts.Admin,
			publicKeys: opts.PublicKeys,
		}
		return nil
	}); err != nil {
		return nil, sqlite.WrapDbErr(err)
	}

	return user, nil
}

// DeleteUser deletes a user.
func (d *SqliteAuthStore) DeleteUser(ctx context.Context, username string) error {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return err
	}

	return sqlite.WrapDbErr(
		sqlite.WrapTx(d.db.DBx(), ctx, func(tx *sqlx.Tx) error {
			_, err := tx.Exec("DELETE FROM user WHERE username = ?", username)
			return err
		}),
	)
}

// SetUsername sets the username of a user.
func (d *SqliteAuthStore) SetUsername(ctx context.Context, username string, newUsername string) error {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return err
	}

	return sqlite.WrapDbErr(
		sqlite.WrapTx(d.db.DBx(), ctx, func(tx *sqlx.Tx) error {
			_, err := tx.Exec("UPDATE user SET username = ? WHERE username = ?", newUsername, username)
			return err
		}),
	)
}

// SetAdmin sets the admin flag of a user.
func (d *SqliteAuthStore) SetAdmin(ctx context.Context, username string, admin bool) error {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return err
	}

	return sqlite.WrapDbErr(
		sqlite.WrapTx(d.db.DBx(), ctx, func(tx *sqlx.Tx) error {
			_, err := tx.Exec("UPDATE user SET admin = ? WHERE username = ?", admin, username)
			return err
		}),
	)
}

// User finds a user by username.
func (d *SqliteAuthStore) User(ctx context.Context, username string) (auth.User, error) {
	return d.user(ctx, username, nil)
}

// UserByPublicKey finds a user by public key.
func (d *SqliteAuthStore) UserByPublicKey(ctx context.Context, pk ssh.PublicKey) (auth.User, error) {
	return d.user(ctx, "", pk)
}

func (d *SqliteAuthStore) user(ctx context.Context, username string, pk ssh.PublicKey) (*User, error) {
	if username == "" && pk == nil {
		return nil, errors.New("username or public key must be provided")
	}

	user := &User{
		username:   username,
		publicKeys: make([]ssh.PublicKey, 0),
	}
	if err := sqlite.WrapTx(d.db.DBx(), ctx, func(tx *sqlx.Tx) error {
		if username == "" {
			row := tx.QueryRow(`SELECT user.username, user.admin
			FROM public_key
			INNER JOIN user ON user.id = public_key.user_id
			WHERE public_key.public_key = ?;`, sshutils.MarshalAuthorizedKey(pk))
			if err := row.Scan(&user.username, &user.isAdmin); err != nil {
				return err
			}
		} else {
			row := tx.QueryRow(`SELECT user.admin
			FROM user
			WHERE user.username = ?;`, username)
			if err := row.Scan(&user.isAdmin); err != nil {
				return err
			}
		}

		rows, err := tx.Query(`SELECT public_key.public_key
			FROM public_key
			INNER JOIN user ON user.id = public_key.user_id
			WHERE user.username = ?;`, user.username)
		if err != nil {
			return err
		}

		for rows.Next() {
			var ak string
			if err := rows.Scan(&ak); err != nil {
				return err
			}

			if pk, _, err := sshutils.ParseAuthorizedKey(ak); err == nil {
				user.publicKeys = append(user.publicKeys, pk)
			}
		}

		return nil
	}); err != nil {
		return nil, sqlite.WrapDbErr(err)
	}

	return user, nil
}

// Users returns all users.
//
// TODO: pagination
func (d *SqliteAuthStore) Users(ctx context.Context) ([]string, error) {
	var users []string
	if err := sqlite.WrapTx(d.db.DBx(), ctx, func(tx *sqlx.Tx) error {
		return tx.Select(&users, "SELECT username FROM user")
	}); err != nil {
		return nil, sqlite.WrapDbErr(err)
	}

	return users, nil
}

// AddPublicKey adds a public key to a user.
//
// It implements backend.Backend.
func (d *SqliteAuthStore) AddPublicKey(ctx context.Context, username string, pk ssh.PublicKey) error {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return err
	}

	return sqlite.WrapDbErr(
		sqlite.WrapTx(d.db.DBx(), ctx, func(tx *sqlx.Tx) error {
			var userID int
			if err := tx.Get(&userID, "SELECT id FROM user WHERE username = ?", username); err != nil {
				return err
			}

			_, err := tx.Exec(`INSERT INTO public_key (user_id, public_key, updated_at)
			VALUES (?, ?, CURRENT_TIMESTAMP);`, userID, sshutils.MarshalAuthorizedKey(pk))
			return err
		}),
	)
}

// RemovePublicKey removes a public key from a user.
//
// It implements backend.Backend.
func (d *SqliteAuthStore) RemovePublicKey(ctx context.Context, username string, pk ssh.PublicKey) error {
	return sqlite.WrapDbErr(
		sqlite.WrapTx(d.db.DBx(), ctx, func(tx *sqlx.Tx) error {
			_, err := tx.Exec(`DELETE FROM public_key
			WHERE user_id = (SELECT id FROM user WHERE username = ?)
			AND public_key = ?;`, username, sshutils.MarshalAuthorizedKey(pk))
			return err
		}),
	)
}

// ListPublicKeys lists the public keys of a user.
func (d *SqliteAuthStore) ListPublicKeys(ctx context.Context, username string) ([]ssh.PublicKey, error) {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return nil, err
	}

	keys := make([]ssh.PublicKey, 0)
	if err := sqlite.WrapTx(d.db.DBx(), ctx, func(tx *sqlx.Tx) error {
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
		return nil, sqlite.WrapDbErr(err)
	}

	return keys, nil
}
