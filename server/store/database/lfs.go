package database

import (
	"context"
	"strings"

	"github.com/charmbracelet/soft-serve/server/db"
	"github.com/charmbracelet/soft-serve/server/db/models"
	"github.com/charmbracelet/soft-serve/server/store"
)

type lfsStore struct{}

var _ store.LFSStore = (*lfsStore)(nil)

func sanitizePath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "/")
	return path
}

// CreateLFSLockForUser implements store.LFSStore.
func (*lfsStore) CreateLFSLockForUser(ctx context.Context, tx db.Handler, repoID int64, userID int64, path string, refname string) error {
	path = sanitizePath(path)
	query := tx.Rebind(`INSERT INTO lfs_locks (repo_id, user_id, path, refname, updated_at)
		VALUES (
			?,
			?,
			?,
			?,
			CURRENT_TIMESTAMP
		);
	`)
	_, err := tx.ExecContext(ctx, query, repoID, userID, path, refname)
	return db.WrapError(err)
}

// GetLFSLocks implements store.LFSStore.
func (*lfsStore) GetLFSLocks(ctx context.Context, tx db.Handler, repoID int64, page int, limit int) ([]models.LFSLock, error) {
	if page <= 0 {
		page = 1
	}

	var locks []models.LFSLock
	query := tx.Rebind(`
		SELECT *
		FROM lfs_locks
		WHERE repo_id = ?
		ORDER BY updated_at DESC
		LIMIT ? OFFSET ?;
	`)
	err := tx.SelectContext(ctx, &locks, query, repoID, limit, (page-1)*limit)
	return locks, db.WrapError(err)
}

func (s *lfsStore) GetLFSLocksWithCount(ctx context.Context, tx db.Handler, repoID int64, page int, limit int) ([]models.LFSLock, int64, error) {
	locks, err := s.GetLFSLocks(ctx, tx, repoID, page, limit)
	if err != nil {
		return nil, 0, err
	}

	var count int64
	query := tx.Rebind(`
		SELECT COUNT(*)
		FROM lfs_locks
		WHERE repo_id = ?;
	`)
	err = tx.GetContext(ctx, &count, query, repoID)
	if err != nil {
		return nil, 0, db.WrapError(err)
	}

	return locks, count, nil
}

// GetLFSLocksForUser implements store.LFSStore.
func (*lfsStore) GetLFSLocksForUser(ctx context.Context, tx db.Handler, repoID int64, userID int64) ([]models.LFSLock, error) {
	var locks []models.LFSLock
	query := tx.Rebind(`
		SELECT *
		FROM lfs_locks
		WHERE repo_id = ? AND user_id = ?;
	`)
	err := tx.SelectContext(ctx, &locks, query, repoID, userID)
	return locks, db.WrapError(err)
}

// GetLFSLocksForPath implements store.LFSStore.
func (*lfsStore) GetLFSLockForPath(ctx context.Context, tx db.Handler, repoID int64, path string) (models.LFSLock, error) {
	path = sanitizePath(path)
	var lock models.LFSLock
	query := tx.Rebind(`
		SELECT *
		FROM lfs_locks
		WHERE repo_id = ? AND path = ?;
	`)
	err := tx.GetContext(ctx, &lock, query, repoID, path)
	return lock, db.WrapError(err)
}

// GetLFSLockForUserPath implements store.LFSStore.
func (*lfsStore) GetLFSLockForUserPath(ctx context.Context, tx db.Handler, repoID int64, userID int64, path string) (models.LFSLock, error) {
	path = sanitizePath(path)
	var lock models.LFSLock
	query := tx.Rebind(`
		SELECT *
		FROM lfs_locks
		WHERE repo_id = ? AND user_id = ? AND path = ?;
	`)
	err := tx.GetContext(ctx, &lock, query, repoID, userID, path)
	return lock, db.WrapError(err)
}

// GetLFSLockByID implements store.LFSStore.
func (*lfsStore) GetLFSLockByID(ctx context.Context, tx db.Handler, id int64) (models.LFSLock, error) {
	var lock models.LFSLock
	query := tx.Rebind(`
		SELECT *
		FROM lfs_locks
		WHERE lfs_locks.id = ?;
	`)
	err := tx.GetContext(ctx, &lock, query, id)
	return lock, db.WrapError(err)
}

// GetLFSLockForUserByID implements store.LFSStore.
func (*lfsStore) GetLFSLockForUserByID(ctx context.Context, tx db.Handler, repoID int64, userID int64, id int64) (models.LFSLock, error) {
	var lock models.LFSLock
	query := tx.Rebind(`
		SELECT *
		FROM lfs_locks
		WHERE id = ? AND user_id = ? AND repo_id = ?;
	`)
	err := tx.GetContext(ctx, &lock, query, id, userID, repoID)
	return lock, db.WrapError(err)
}

// DeleteLFSLockForUserByID implements store.LFSStore.
func (*lfsStore) DeleteLFSLockForUserByID(ctx context.Context, tx db.Handler, repoID int64, userID int64, id int64) error {
	query := tx.Rebind(`
		DELETE FROM lfs_locks
		WHERE repo_id = ? AND user_id = ? AND id = ?;
	`)
	_, err := tx.ExecContext(ctx, query, repoID, userID, id)
	return db.WrapError(err)
}

// DeleteLFSLock implements store.LFSStore.
func (*lfsStore) DeleteLFSLock(ctx context.Context, tx db.Handler, repoID int64, id int64) error {
	query := tx.Rebind(`
		DELETE FROM lfs_locks
		WHERE repo_id = ? AND id = ?;
	`)
	_, err := tx.ExecContext(ctx, query, repoID, id)
	return db.WrapError(err)
}

// CreateLFSObject implements store.LFSStore.
func (*lfsStore) CreateLFSObject(ctx context.Context, tx db.Handler, repoID int64, oid string, size int64) error {
	query := tx.Rebind(`INSERT INTO lfs_objects (repo_id, oid, size, updated_at) VALUES (?, ?, ?, CURRENT_TIMESTAMP);`)
	_, err := tx.ExecContext(ctx, query, repoID, oid, size)
	return db.WrapError(err)
}

// DeleteLFSObjectByOid implements store.LFSStore.
func (*lfsStore) DeleteLFSObjectByOid(ctx context.Context, tx db.Handler, repoID int64, oid string) error {
	query := tx.Rebind(`DELETE FROM lfs_objects WHERE repo_id = ? AND oid = ?;`)
	_, err := tx.ExecContext(ctx, query, repoID, oid)
	return db.WrapError(err)
}

// GetLFSObjectByOid implements store.LFSStore.
func (*lfsStore) GetLFSObjectByOid(ctx context.Context, tx db.Handler, repoID int64, oid string) (models.LFSObject, error) {
	var obj models.LFSObject
	query := tx.Rebind(`SELECT * FROM lfs_objects WHERE repo_id = ? AND oid = ?;`)
	err := tx.GetContext(ctx, &obj, query, repoID, oid)
	return obj, db.WrapError(err)
}

// GetLFSObjects implements store.LFSStore.
func (*lfsStore) GetLFSObjects(ctx context.Context, tx db.Handler, repoID int64) ([]models.LFSObject, error) {
	var objs []models.LFSObject
	query := tx.Rebind(`SELECT * FROM lfs_objects WHERE repo_id = ?;`)
	err := tx.SelectContext(ctx, &objs, query, repoID)
	return objs, db.WrapError(err)
}

// GetLFSObjectsByName implements store.LFSStore.
func (*lfsStore) GetLFSObjectsByName(ctx context.Context, tx db.Handler, name string) ([]models.LFSObject, error) {
	var objs []models.LFSObject
	query := tx.Rebind(`
		SELECT lfs_objects.*
		FROM lfs_objects
		INNER JOIN repos ON lfs_objects.repo_id = repos.id
		WHERE repos.name = ?;
	`)
	err := tx.SelectContext(ctx, &objs, query, name)
	return objs, db.WrapError(err)
}
