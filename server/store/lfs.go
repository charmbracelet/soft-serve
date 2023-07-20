package store

import (
	"context"

	"github.com/charmbracelet/soft-serve/server/db"
	"github.com/charmbracelet/soft-serve/server/db/models"
)

// LFSStore is the interface for the LFS store.
type LFSStore interface {
	CreateLFSObject(ctx context.Context, h db.Handler, repoID int64, oid string, size int64) error
	GetLFSObjectByOid(ctx context.Context, h db.Handler, repoID int64, oid string) (models.LFSObject, error)
	GetLFSObjects(ctx context.Context, h db.Handler, repoID int64) ([]models.LFSObject, error)
	GetLFSObjectsByName(ctx context.Context, h db.Handler, name string) ([]models.LFSObject, error)
	DeleteLFSObjectByOid(ctx context.Context, h db.Handler, repoID int64, oid string) error

	CreateLFSLockForUser(ctx context.Context, h db.Handler, repoID int64, userID int64, path string) error
	GetLFSLocks(ctx context.Context, h db.Handler, repoID int64) ([]models.LFSLock, error)
	GetLFSLocksForUser(ctx context.Context, h db.Handler, repoID int64, userID int64) ([]models.LFSLock, error)
	GetLFSLocksForPath(ctx context.Context, h db.Handler, repoID int64, path string) ([]models.LFSLock, error)
	GetLFSLockForUserPath(ctx context.Context, h db.Handler, repoID int64, userID int64, path string) (models.LFSLock, error)
	GetLFSLockByID(ctx context.Context, h db.Handler, id string) (models.LFSLock, error)
	GetLFSLockForUserByID(ctx context.Context, h db.Handler, userID int64, id string) (models.LFSLock, error)
	DeleteLFSLockForUserByID(ctx context.Context, h db.Handler, userID int64, id string) error
}
