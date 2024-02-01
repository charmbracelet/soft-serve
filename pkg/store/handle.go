package store

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
)

// HandleStore is a store for username handles.
type HandleStore interface {
	GetHandleByID(ctx context.Context, h db.Handler, id int64) (models.Handle, error)
	GetHandleByHandle(ctx context.Context, h db.Handler, handle string) (models.Handle, error)
	GetHandleByUserID(ctx context.Context, h db.Handler, userID int64) (models.Handle, error)
	ListHandlesForIDs(ctx context.Context, h db.Handler, ids []int64) ([]models.Handle, error)
	UpdateHandle(ctx context.Context, h db.Handler, id int64, handle string) error
	CreateHandle(ctx context.Context, h db.Handler, handle string) (int64, error)
	DeleteHandle(ctx context.Context, h db.Handler, id int64) error
}
