package store

import (
	"context"

	"github.com/charmbracelet/soft-serve/server/db"
	"github.com/charmbracelet/soft-serve/server/db/models"
	"golang.org/x/crypto/ssh"
)

// UserStore is an interface for managing users.
type UserStore interface {
	GetUserByID(ctx context.Context, h db.Handler, id int64) (models.User, error)
	FindUserByUsername(ctx context.Context, h db.Handler, username string) (models.User, error)
	FindUserByPublicKey(ctx context.Context, h db.Handler, pk ssh.PublicKey) (models.User, error)
	GetAllUsers(ctx context.Context, h db.Handler) ([]models.User, error)
	CreateUser(ctx context.Context, h db.Handler, username string, isAdmin bool, pks []ssh.PublicKey) error
	DeleteUserByUsername(ctx context.Context, h db.Handler, username string) error
	SetUsernameByUsername(ctx context.Context, h db.Handler, username string, newUsername string) error
	SetAdminByUsername(ctx context.Context, h db.Handler, username string, isAdmin bool) error
	AddPublicKeyByUsername(ctx context.Context, h db.Handler, username string, pk ssh.PublicKey) error
	RemovePublicKeyByUsername(ctx context.Context, h db.Handler, username string, pk ssh.PublicKey) error
	ListPublicKeysByUserID(ctx context.Context, h db.Handler, id int64) ([]ssh.PublicKey, error)
	ListPublicKeysByUsername(ctx context.Context, h db.Handler, username string) ([]ssh.PublicKey, error)
	SetUserPassword(ctx context.Context, h db.Handler, userID int64, password string) error
	SetUserPasswordByUsername(ctx context.Context, h db.Handler, username string, password string) error
}
