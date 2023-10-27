package store

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
)

// RepositoryStore is an interface for managing repositories.
type RepositoryStore interface {
	GetRepoByName(ctx context.Context, h db.Handler, name string) (models.Repo, error)
	GetAllRepos(ctx context.Context, h db.Handler) ([]models.Repo, error)
	GetUserRepos(ctx context.Context, h db.Handler, userID int64) ([]models.Repo, error)
	CreateRepo(ctx context.Context, h db.Handler, name string, userID int64, projectName string, description string, isPrivate bool, isHidden bool, isMirror bool) error
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
