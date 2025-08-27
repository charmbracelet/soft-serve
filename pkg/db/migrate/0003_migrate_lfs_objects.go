package migrate

import (
	"context"
	"os"
	"path/filepath"
	"strconv"

	"github.com/charmbracelet/log/v2"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
)

const (
	migrateLfsObjectsName    = "migrate_lfs_objects"
	migrateLfsObjectsVersion = 3
)

// Correct LFS objects relative path.
// From OID[:2]/OID[2:4]/OID[4:] to OID[:2]/OID[2:4]/OID
// See: https://github.com/git-lfs/git-lfs/blob/main/docs/spec.md#intercepting-git
var migrateLfsObjects = Migration{
	Name:    migrateLfsObjectsName,
	Version: migrateLfsObjectsVersion,
	Migrate: func(ctx context.Context, tx *db.Tx) error {
		cfg := config.FromContext(ctx)
		logger := log.FromContext(ctx).WithPrefix("migrate_lfs_objects")

		var repoIDs []int64
		if err := tx.Select(&repoIDs, "SELECT id FROM repos"); err != nil {
			return err //nolint:wrapcheck
		}
		for _, r := range repoIDs {
			var objs []models.LFSObject
			if err := tx.Select(&objs, "SELECT * FROM lfs_objects WHERE repo_id = ?", r); err != nil {
				return err //nolint:wrapcheck
			}
			objsp := filepath.Join(cfg.DataPath, "lfs", strconv.FormatInt(r, 10), "objects")
			for _, obj := range objs {
				oldpath := filepath.Join(objsp, badRelativePath(obj.Oid))
				newpath := filepath.Join(objsp, goodRelativePath(obj.Oid))
				if _, err := os.Stat(oldpath); err == nil {
					if err := os.Rename(oldpath, newpath); err != nil {
						logger.Error("rename lfs object", "oldpath", oldpath, "newpath", newpath, "err", err)
						continue
					}
				}
			}
		}
		return nil
	},
	Rollback: func(context.Context, *db.Tx) error {
		return nil
	},
}

func goodRelativePath(oid string) string {
	if len(oid) < 5 {
		return oid
	}
	return filepath.Join(oid[:2], oid[2:4], oid)
}

func badRelativePath(oid string) string {
	if len(oid) < 5 {
		return oid
	}
	return filepath.Join(oid[:2], oid[2:4], oid[4:])
}
