package proto

import (
	"errors"
)

var (
	// ErrUnauthorized is returned when the user is not authorized to perform action.
	ErrUnauthorized = errors.New("unauthorized")
	// ErrInvalidRemote is returned when a repository import remote is invalid.
	ErrInvalidRemote = errors.New("remote must be a network URL")
	// ErrImportFailed is returned when cloning an import remote fails. It
	// carries no detail on purpose: the git error would tell the importing
	// user whether a host is up, which port answered, and what it said.
	ErrImportFailed = errors.New("import failed")
	// ErrImportsDisabled is returned when the server has turned imports off,
	// which it does when the installed git is too old to enforce the remote
	// validation. The reason stays in the server log.
	ErrImportsDisabled = errors.New("imports are disabled on this server")
	// ErrFileNotFound is returned when the file is not found.
	ErrFileNotFound = errors.New("file not found")
	// ErrRepoNotFound is returned when a repository is not found.
	ErrRepoNotFound = errors.New("repository not found")
	// ErrRepoExist is returned when a repository already exists.
	ErrRepoExist = errors.New("repository already exists")
	// ErrUserNotFound is returned when a user is not found.
	ErrUserNotFound = errors.New("user not found")
	// ErrUserOwnsRepos is returned when deleting a user who still owns
	// repositories.
	ErrUserOwnsRepos = errors.New("user still owns repositories; delete them first")
	// ErrTokenNotFound is returned when a token is not found.
	ErrTokenNotFound = errors.New("token not found")
	// ErrTokenExpired is returned when a token is expired.
	ErrTokenExpired = errors.New("token expired")
	// ErrCollaboratorNotFound is returned when a collaborator is not found.
	ErrCollaboratorNotFound = errors.New("collaborator not found")
	// ErrCollaboratorExist is returned when a collaborator already exists.
	ErrCollaboratorExist = errors.New("collaborator already exists")
	// ErrExceedsAccessLevel is returned when an action would grant or revoke
	// access above the caller's own access level.
	ErrExceedsAccessLevel = errors.New("cannot change access above your own access level")
)
