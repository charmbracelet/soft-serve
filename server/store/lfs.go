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

	CreateLFSLockForUser(ctx context.Context, h db.Handler, repoID int64, userID int64, path string, refname string) error
	GetLFSLocks(ctx context.Context, h db.Handler, repoID int64, page int, limit int) ([]models.LFSLock, error)
	GetLFSLocksWithCount(ctx context.Context, h db.Handler, repoID int64, page int, limit int) ([]models.LFSLock, int64, error)
	GetLFSLocksForUser(ctx context.Context, h db.Handler, repoID int64, userID int64) ([]models.LFSLock, error)
	GetLFSLockForPath(ctx context.Context, h db.Handler, repoID int64, path string) (models.LFSLock, error)
	GetLFSLockForUserPath(ctx context.Context, h db.Handler, repoID int64, userID int64, path string) (models.LFSLock, error)
	GetLFSLockByID(ctx context.Context, h db.Handler, id int64) (models.LFSLock, error)
	GetLFSLockForUserByID(ctx context.Context, h db.Handler, repoID int64, userID int64, id int64) (models.LFSLock, error)
	DeleteLFSLock(ctx context.Context, h db.Handler, repoID int64, id int64) error
	DeleteLFSLockForUserByID(ctx context.Context, h db.Handler, repoID int64, userID int64, id int64) error
}
