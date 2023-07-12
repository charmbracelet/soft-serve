package store

import (
	"context"

	"github.com/charmbracelet/soft-serve/server/db"
	"github.com/charmbracelet/soft-serve/server/db/models"
	"golang.org/x/crypto/ssh"
)

// SettingStore is an interface for managing settings.
type SettingStore interface {
	GetAnonAccess(ctx context.Context, tx *db.Tx) (AccessLevel, error)
	SetAnonAccess(ctx context.Context, tx *db.Tx, level AccessLevel) error
	GetAllowKeylessAccess(ctx context.Context, tx *db.Tx) (bool, error)
	SetAllowKeylessAccess(ctx context.Context, tx *db.Tx, allow bool) error
}

// RepositoryStore is an interface for managing repositories.
type RepositoryStore interface {
	GetRepoByName(ctx context.Context, tx *db.Tx, name string) (models.Repo, error)
	GetAllRepos(ctx context.Context, tx *db.Tx) ([]models.Repo, error)
	CreateRepo(ctx context.Context, tx *db.Tx, name string, projectName string, description string, isPrivate bool, isHidden bool, isMirror bool) error
	DeleteRepoByName(ctx context.Context, tx *db.Tx, name string) error
	SetRepoNameByName(ctx context.Context, tx *db.Tx, name string, newName string) error

	GetRepoProjectNameByName(ctx context.Context, tx *db.Tx, name string) (string, error)
	SetRepoProjectNameByName(ctx context.Context, tx *db.Tx, name string, projectName string) error
	GetRepoDescriptionByName(ctx context.Context, tx *db.Tx, name string) (string, error)
	SetRepoDescriptionByName(ctx context.Context, tx *db.Tx, name string, description string) error
	GetRepoIsPrivateByName(ctx context.Context, tx *db.Tx, name string) (bool, error)
	SetRepoIsPrivateByName(ctx context.Context, tx *db.Tx, name string, isPrivate bool) error
	GetRepoIsHiddenByName(ctx context.Context, tx *db.Tx, name string) (bool, error)
	SetRepoIsHiddenByName(ctx context.Context, tx *db.Tx, name string, isHidden bool) error
	GetRepoIsMirrorByName(ctx context.Context, tx *db.Tx, name string) (bool, error)
}

// UserStore is an interface for managing users.
type UserStore interface {
	FindUserByUsername(ctx context.Context, tx *db.Tx, username string) (models.User, error)
	FindUserByPublicKey(ctx context.Context, tx *db.Tx, pk ssh.PublicKey) (models.User, error)
	GetAllUsers(ctx context.Context, tx *db.Tx) ([]models.User, error)
	CreateUser(ctx context.Context, tx *db.Tx, username string, isAdmin bool, pks []ssh.PublicKey) error
	DeleteUserByUsername(ctx context.Context, tx *db.Tx, username string) error
	SetUsernameByUsername(ctx context.Context, tx *db.Tx, username string, newUsername string) error
	SetAdminByUsername(ctx context.Context, tx *db.Tx, username string, isAdmin bool) error
	AddPublicKeyByUsername(ctx context.Context, tx *db.Tx, username string, pk ssh.PublicKey) error
	RemovePublicKeyByUsername(ctx context.Context, tx *db.Tx, username string, pk ssh.PublicKey) error
	ListPublicKeysByUserID(ctx context.Context, tx *db.Tx, id int64) ([]ssh.PublicKey, error)
	ListPublicKeysByUsername(ctx context.Context, tx *db.Tx, username string) ([]ssh.PublicKey, error)
}

// CollaboratorStore is an interface for managing collaborators.
type CollaboratorStore interface {
	GetCollabByUsernameAndRepo(ctx context.Context, tx *db.Tx, username string, repo string) (models.Collab, error)
	AddCollabByUsernameAndRepo(ctx context.Context, tx *db.Tx, username string, repo string) error
	RemoveCollabByUsernameAndRepo(ctx context.Context, tx *db.Tx, username string, repo string) error
	ListCollabsByRepo(ctx context.Context, tx *db.Tx, repo string) ([]models.Collab, error)
	ListCollabsByRepoAsUsers(ctx context.Context, tx *db.Tx, repo string) ([]models.User, error)
}

// Store is an interface for managing repositories, users, and settings.
type Store interface {
	RepositoryStore
	UserStore
	CollaboratorStore
	SettingStore
}
