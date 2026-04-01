package database

import (
	"context"
	"errors"
	"strings"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/sshutils"
	"github.com/charmbracelet/soft-serve/pkg/store"
	"github.com/charmbracelet/soft-serve/pkg/utils"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ssh"
)

const sqliteMaxUserPlaceholders = 999

type userStore struct{}

var _ store.UserStore = (*userStore)(nil)

func validateBcryptHash(password string) error {
	if _, err := bcrypt.Cost([]byte(password)); err != nil {
		return errors.New("password must be bcrypt hashed")
	}
	return nil
}

func (*userStore) AddPublicKeyByUsername(ctx context.Context, tx db.Handler, username string, pk ssh.PublicKey) error {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return err
	}

	var userID int64
	if err := tx.GetContext(ctx, &userID, tx.Rebind(`SELECT id FROM users WHERE username = ?`), username); err != nil {
		return db.WrapError(err)
	}

	query := tx.Rebind(`INSERT INTO public_keys (user_id, public_key, updated_at)
			VALUES (?, ?, CURRENT_TIMESTAMP);`)
	ak := sshutils.MarshalAuthorizedKey(pk)
	_, err := tx.ExecContext(ctx, query, userID, ak)

	return db.WrapError(err)
}

func (*userStore) CreateUser(ctx context.Context, tx db.Handler, username string, isAdmin bool, pks []ssh.PublicKey) error {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return err
	}

	query := tx.Rebind(`INSERT INTO users (username, admin, updated_at)
			VALUES (?, ?, CURRENT_TIMESTAMP) RETURNING id;`)

	var userID int64
	if err := tx.GetContext(ctx, &userID, query, username, isAdmin); err != nil {
		return db.WrapError(err)
	}

	for i := 0; i < len(pks); i += sqliteMaxUserPlaceholders {
		end := i + len(pks)
		if end-i > sqliteMaxUserPlaceholders {
			end = i + sqliteMaxUserPlaceholders
		}
		batch := pks[i:end]

		var pb strings.Builder
		args := make([]interface{}, 0, len(batch)*2)
		for j := range batch {
			if j > 0 {
				pb.WriteString(", ")
			}
			pb.WriteString("(?, ?, CURRENT_TIMESTAMP)")
			ak := sshutils.MarshalAuthorizedKey(batch[j])
			args = append(args, userID, ak)
		}
		query := tx.Rebind(`INSERT INTO public_keys (user_id, public_key, updated_at) VALUES ` + pb.String())
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return db.WrapError(err)
		}
	}

	return nil
}

func (*userStore) DeleteUserByUsername(ctx context.Context, tx db.Handler, username string) error {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return err
	}

	query := tx.Rebind(`DELETE FROM users WHERE username = ?;`)
	_, err := tx.ExecContext(ctx, query, username)
	return db.WrapError(err)
}

func (*userStore) GetUserByID(ctx context.Context, tx db.Handler, id int64) (models.User, error) {
	var m models.User
	query := tx.Rebind(`SELECT * FROM users WHERE id = ?;`)
	err := tx.GetContext(ctx, &m, query, id)
	return m, db.WrapError(err)
}

func (*userStore) FindUserByPublicKey(ctx context.Context, tx db.Handler, pk ssh.PublicKey) (models.User, error) {
	var m models.User
	query := tx.Rebind(`SELECT users.*
			FROM users
			INNER JOIN public_keys ON users.id = public_keys.user_id
			WHERE public_keys.public_key = ?;`)
	err := tx.GetContext(ctx, &m, query, sshutils.MarshalAuthorizedKey(pk))
	return m, db.WrapError(err)
}

func (*userStore) FindUserByUsername(ctx context.Context, tx db.Handler, username string) (models.User, error) {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return models.User{}, err
	}

	var m models.User
	query := tx.Rebind(`SELECT * FROM users WHERE username = ?;`)
	err := tx.GetContext(ctx, &m, query, username)
	return m, db.WrapError(err)
}

func (*userStore) FindUserByAccessToken(ctx context.Context, tx db.Handler, token string) (models.User, error) {
	var m models.User
	query := tx.Rebind(`SELECT users.*
			FROM users
			INNER JOIN access_tokens ON users.id = access_tokens.user_id
			WHERE access_tokens.token = ?;`)
	err := tx.GetContext(ctx, &m, query, token)
	return m, db.WrapError(err)
}

func (*userStore) GetAllUsers(ctx context.Context, tx db.Handler) ([]models.User, error) {
	var ms []models.User
	query := tx.Rebind("SELECT * FROM users LIMIT 10000;")
	err := tx.SelectContext(ctx, &ms, query)
	return ms, db.WrapError(err)
}

func (*userStore) ListPublicKeysByUserID(ctx context.Context, tx db.Handler, id int64) ([]ssh.PublicKey, error) {
	var aks []string
	query := tx.Rebind(`SELECT public_key FROM public_keys
			WHERE user_id = ?
			ORDER BY public_keys.id ASC;`)
	err := tx.SelectContext(ctx, &aks, query, id)
	if err != nil {
		return nil, db.WrapError(err)
	}

	pks := make([]ssh.PublicKey, len(aks))
	for i, ak := range aks {
		pk, _, err := sshutils.ParseAuthorizedKey(ak)
		if err != nil {
			return nil, err
		}
		pks[i] = pk
	}

	return pks, nil
}

func (*userStore) ListPublicKeysByUsername(ctx context.Context, tx db.Handler, username string) ([]ssh.PublicKey, error) {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return nil, err
	}

	var aks []string
	query := tx.Rebind(`SELECT public_key FROM public_keys
			INNER JOIN users ON users.id = public_keys.user_id
			WHERE users.username = ?
			ORDER BY public_keys.id ASC;`)
	err := tx.SelectContext(ctx, &aks, query, username)
	if err != nil {
		return nil, db.WrapError(err)
	}

	pks := make([]ssh.PublicKey, len(aks))
	for i, ak := range aks {
		pk, _, err := sshutils.ParseAuthorizedKey(ak)
		if err != nil {
			return nil, err
		}
		pks[i] = pk
	}

	return pks, nil
}

func (*userStore) RemovePublicKeyByUsername(ctx context.Context, tx db.Handler, username string, pk ssh.PublicKey) error {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return err
	}

	query := tx.Rebind(`DELETE FROM public_keys
			WHERE user_id = (SELECT id FROM users WHERE username = ?)
			AND public_key = ?;`)
	_, err := tx.ExecContext(ctx, query, username, sshutils.MarshalAuthorizedKey(pk))
	return db.WrapError(err)
}

func (*userStore) SetAdminByUsername(ctx context.Context, tx db.Handler, username string, isAdmin bool) error {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return err
	}

	query := tx.Rebind(`UPDATE users SET admin = ? WHERE username = ?;`)
	_, err := tx.ExecContext(ctx, query, isAdmin, username)
	return db.WrapError(err)
}

func (*userStore) SetUsernameByUsername(ctx context.Context, tx db.Handler, username string, newUsername string) error {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return err
	}

	newUsername = strings.ToLower(newUsername)
	if err := utils.ValidateUsername(newUsername); err != nil {
		return err
	}

	query := tx.Rebind(`UPDATE users SET username = ? WHERE username = ?;`)
	_, err := tx.ExecContext(ctx, query, newUsername, username)
	return db.WrapError(err)
}

func (*userStore) SetUserPassword(ctx context.Context, tx db.Handler, userID int64, password string) error {
	if err := validateBcryptHash(password); err != nil {
		return err
	}
	query := tx.Rebind(`UPDATE users SET password = ? WHERE id = ?;`)
	_, err := tx.ExecContext(ctx, query, password, userID)
	return db.WrapError(err)
}

func (*userStore) SetUserPasswordByUsername(ctx context.Context, tx db.Handler, username string, password string) error {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return err
	}

	if err := validateBcryptHash(password); err != nil {
		return err
	}

	query := tx.Rebind(`UPDATE users SET password = ? WHERE username = ?;`)
	_, err := tx.ExecContext(ctx, query, password, username)
	return db.WrapError(err)
}
