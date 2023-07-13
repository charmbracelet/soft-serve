package database

import (
	"context"
	"strings"

	"github.com/charmbracelet/soft-serve/server/db"
	"github.com/charmbracelet/soft-serve/server/db/models"
	"github.com/charmbracelet/soft-serve/server/sshutils"
	"github.com/charmbracelet/soft-serve/server/store"
	"github.com/charmbracelet/soft-serve/server/utils"
	"golang.org/x/crypto/ssh"
)

type userStore struct{}

var _ store.UserStore = (*userStore)(nil)

// AddPublicKeyByUsername implements store.UserStore.
func (*userStore) AddPublicKeyByUsername(ctx context.Context, tx *db.Tx, username string, pk ssh.PublicKey) error {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return err
	}

	var userID int64
	if err := tx.GetContext(ctx, &userID, tx.Rebind(`SELECT id FROM users WHERE username = ?`), username); err != nil {
		return err
	}

	query := tx.Rebind(`INSERT INTO public_keys (user_id, public_key, updated_at)
			VALUES (?, ?, CURRENT_TIMESTAMP);`)
	ak := sshutils.MarshalAuthorizedKey(pk)
	_, err := tx.ExecContext(ctx, query, userID, ak)

	return err
}

// CreateUser implements store.UserStore.
func (*userStore) CreateUser(ctx context.Context, tx *db.Tx, username string, isAdmin bool, pks []ssh.PublicKey) error {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return err
	}

	query := tx.Rebind(`INSERT INTO users (username, admin, updated_at)
			VALUES (?, ?, CURRENT_TIMESTAMP);`)
	result, err := tx.ExecContext(ctx, query, username, isAdmin)
	if err != nil {
		return err
	}

	userID, err := result.LastInsertId()
	if err != nil {
		return err
	}

	for _, pk := range pks {
		query := tx.Rebind(`INSERT INTO public_keys (user_id, public_key, updated_at)
			VALUES (?, ?, CURRENT_TIMESTAMP);`)
		ak := sshutils.MarshalAuthorizedKey(pk)
		_, err := tx.ExecContext(ctx, query, userID, ak)
		if err != nil {
			return err
		}
	}

	return nil
}

// DeleteUserByUsername implements store.UserStore.
func (*userStore) DeleteUserByUsername(ctx context.Context, tx *db.Tx, username string) error {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return err
	}

	query := tx.Rebind(`DELETE FROM users WHERE username = ?;`)
	_, err := tx.ExecContext(ctx, query, username)
	return err
}

// FindUserByPublicKey implements store.UserStore.
func (*userStore) FindUserByPublicKey(ctx context.Context, tx *db.Tx, pk ssh.PublicKey) (models.User, error) {
	var m models.User
	query := tx.Rebind(`SELECT users.*
			FROM users
			INNER JOIN public_keys ON users.id = public_keys.user_id
			WHERE public_keys.public_key = ?;`)
	err := tx.GetContext(ctx, &m, query, sshutils.MarshalAuthorizedKey(pk))
	return m, err
}

// FindUserByUsername implements store.UserStore.
func (*userStore) FindUserByUsername(ctx context.Context, tx *db.Tx, username string) (models.User, error) {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return models.User{}, err
	}

	var m models.User
	query := tx.Rebind(`SELECT * FROM users WHERE username = ?;`)
	err := tx.GetContext(ctx, &m, query, username)
	return m, err
}

// GetAllUsers implements store.UserStore.
func (*userStore) GetAllUsers(ctx context.Context, tx *db.Tx) ([]models.User, error) {
	var ms []models.User
	query := tx.Rebind(`SELECT * FROM users;`)
	err := tx.SelectContext(ctx, &ms, query)
	return ms, err
}

// ListPublicKeysByUserID implements store.UserStore..
func (*userStore) ListPublicKeysByUserID(ctx context.Context, tx *db.Tx, id int64) ([]ssh.PublicKey, error) {
	var aks []string
	query := tx.Rebind(`SELECT public_key FROM public_keys
			WHERE user_id = ?
			ORDER BY public_keys.id ASC;`)
	err := tx.SelectContext(ctx, &aks, query, id)
	if err != nil {
		return nil, err
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

// ListPublicKeysByUsername implements store.UserStore.
func (*userStore) ListPublicKeysByUsername(ctx context.Context, tx *db.Tx, username string) ([]ssh.PublicKey, error) {
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
		return nil, err
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

// RemovePublicKeyByUsername implements store.UserStore.
func (*userStore) RemovePublicKeyByUsername(ctx context.Context, tx *db.Tx, username string, pk ssh.PublicKey) error {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return err
	}

	query := tx.Rebind(`DELETE FROM public_keys
			WHERE user_id = (SELECT id FROM users WHERE username = ?)
			AND public_key = ?;`)
	_, err := tx.ExecContext(ctx, query, username, sshutils.MarshalAuthorizedKey(pk))
	return err
}

// SetAdminByUsername implements store.UserStore.
func (*userStore) SetAdminByUsername(ctx context.Context, tx *db.Tx, username string, isAdmin bool) error {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return err
	}

	query := tx.Rebind(`UPDATE users SET admin = ? WHERE username = ?;`)
	_, err := tx.ExecContext(ctx, query, isAdmin, username)
	return err
}

// SetUsernameByUsername implements store.UserStore.
func (*userStore) SetUsernameByUsername(ctx context.Context, tx *db.Tx, username string, newUsername string) error {
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
	return err
}
