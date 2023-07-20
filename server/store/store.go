package store

import (
	"context"

	"github.com/charmbracelet/soft-serve/server/access"
	"github.com/charmbracelet/soft-serve/server/db"
	"github.com/charmbracelet/soft-serve/server/db/models"
	"golang.org/x/crypto/ssh"
)

// SettingStore is an interface for managing settings.
type SettingStore interface {
	GetAnonAccess(ctx context.Context, h db.Handler) (access.AccessLevel, error)
	SetAnonAccess(ctx context.Context, h db.Handler, level access.AccessLevel) error
	GetAllowKeylessAccess(ctx context.Context, h db.Handler) (bool, error)
	SetAllowKeylessAccess(ctx context.Context, h db.Handler, allow bool) error
}

// RepositoryStore is an interface for managing repositories.
type RepositoryStore interface {
	GetRepoByName(ctx context.Context, h db.Handler, name string) (models.Repo, error)
	GetAllRepos(ctx context.Context, h db.Handler) ([]models.Repo, error)
	CreateRepo(ctx context.Context, h db.Handler, name string, projectName string, description string, isPrivate bool, isHidden bool, isMirror bool) error
	DeleteRepoByName(ctx context.Context, h db.Handler, name string) error
	SetRepoNameByName(ctx context.Context, h db.Handler, name string, newName string) error

	GetRepoProjectNameByName(ctx context.Context, h db.Handler, name string) (string, error)
	SetRepoProjectNameByName(ctx context.Context, h db.Handler, name string, projectName string) error
	GetRepoDescriptionByName(ctx context.Context, h db.Handler, name string) (string, error)
	SetRepoDescriptionByName(ctx context.Context, h db.Handler, name string, description string) error
	GetRepoIsPrivateByName(ctx context.Context, h db.Handler, name string) (bool, error)
	SetRepoIsPrivateByName(ctx context.Context, h db.Handler, name string, isPrivate bool) error
	GetRepoIsHiddenByName(ctx context.Context, h db.Handler, name string) (bool, error)
	SetRepoIsHiddenByName(ctx context.Context, h db.Handler, name string, isHidden bool) error
	GetRepoIsMirrorByName(ctx context.Context, h db.Handler, name string) (bool, error)
}

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
}

// CollaboratorStore is an interface for managing collaborators.
type CollaboratorStore interface {
	GetCollabByUsernameAndRepo(ctx context.Context, h db.Handler, username string, repo string) (models.Collab, error)
	AddCollabByUsernameAndRepo(ctx context.Context, h db.Handler, username string, repo string) error
	RemoveCollabByUsernameAndRepo(ctx context.Context, h db.Handler, username string, repo string) error
	ListCollabsByRepo(ctx context.Context, h db.Handler, repo string) ([]models.Collab, error)
	ListCollabsByRepoAsUsers(ctx context.Context, h db.Handler, repo string) ([]models.User, error)
}

// Store is an interface for managing repositories, users, and settings.
type Store interface {
	RepositoryStore
	UserStore
	CollaboratorStore
	SettingStore
	LFSStore
}
