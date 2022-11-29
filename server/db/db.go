package db

import (
	"github.com/charmbracelet/soft-serve/proto"
	"github.com/charmbracelet/soft-serve/server/db/types"
)

// ConfigStore is a configuration database storage.
type ConfigStore interface {
	// Config
	GetConfig() (*types.Config, error)
	SetConfigName(string) error
	SetConfigHost(string) error
	SetConfigPort(int) error
	SetConfigAnonAccess(string) error
	SetConfigAllowKeyless(bool) error
}

// UserStore is a user database storage.
type UserStore interface {
	// Users
	AddUser(name, login, email, password string, isAdmin bool) error
	DeleteUser(int) error
	GetUser(int) (*types.User, error)
	GetUserByLogin(string) (*types.User, error)
	GetUserByEmail(string) (*types.User, error)
	GetUserByPublicKey(string) (*types.User, error)
	SetUserName(*types.User, string) error
	SetUserLogin(*types.User, string) error
	SetUserEmail(*types.User, string) error
	SetUserPassword(*types.User, string) error
	SetUserAdmin(*types.User, bool) error
	CountUsers() (int, error)
}

// PublicKeyStore is a public key database storage.
type PublicKeyStore interface {
	// Public keys
	AddUserPublicKey(*types.User, string) error
	DeleteUserPublicKey(int) error
	GetUserPublicKeys(*types.User) ([]*types.PublicKey, error)
}

// RepoStore is a repository database storage.
type RepoStore interface {
	// Repos
	AddRepo(name, projectName, description string, isPrivate bool) error
	DeleteRepo(string) error
	GetRepo(string) (*types.Repo, error)
	SetRepoProjectName(string, string) error
	SetRepoDescription(string, string) error
	SetRepoPrivate(string, bool) error
}

// CollabStore is a collaborator database storage.
type CollabStore interface {
	// Collaborators
	AddRepoCollab(string, *types.User) error
	DeleteRepoCollab(int, int) error
	ListRepoCollabs(string) ([]*types.User, error)
	ListRepoPublicKeys(string) ([]*types.PublicKey, error)
}

// Store is a database.
type Store interface {
	proto.Provider

	ConfigStore
	UserStore
	PublicKeyStore
	RepoStore
	CollabStore

	// CreateDB creates the database.
	CreateDB() error

	// Close closes the database.
	Close() error
}
