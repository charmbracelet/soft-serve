package store

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
)

// CollaboratorStore is an interface for managing collaborators.
type CollaboratorStore interface {
	GetCollabByUsernameAndRepo(ctx context.Context, h db.Handler, username string, repo string) (models.Collab, error)
	AddCollabByUsernameAndRepo(ctx context.Context, h db.Handler, username string, repo string, level access.AccessLevel) error
	RemoveCollabByUsernameAndRepo(ctx context.Context, h db.Handler, username string, repo string) error
	ListCollabsByRepo(ctx context.Context, h db.Handler, repo string) ([]models.Collab, error)
	ListCollabsByRepoAsUsers(ctx context.Context, h db.Handler, repo string) ([]models.User, error)
}
