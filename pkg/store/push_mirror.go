package store

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
)

// PushMirrorStore is the interface for push mirror storage operations.
type PushMirrorStore interface {
	CreatePushMirror(ctx context.Context, tx db.Handler, repoID int64, name, remoteURL string) error
	DeletePushMirror(ctx context.Context, tx db.Handler, repoID int64, name string) error
	GetPushMirrorsByRepoID(ctx context.Context, tx db.Handler, repoID int64) ([]models.PushMirror, error)
	SetPushMirrorEnabled(ctx context.Context, tx db.Handler, repoID int64, name string, enabled bool) error
}
