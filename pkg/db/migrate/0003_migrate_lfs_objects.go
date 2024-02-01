package migrate

import (
	"context"
	"os"
	"path/filepath"
	"strconv"

	"github.com/charmbracelet/log"
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
	Migrate: func(ctx context.Context, h db.Handler) error {
		cfg := config.FromContext(ctx)
		logger := log.FromContext(ctx).WithPrefix("migrate_lfs_objects")

		var repoIds []int64
		if err := h.Select(&repoIds, "SELECT id FROM repos"); err != nil {
			return err
		}
		for _, r := range repoIds {
			var objs []models.LFSObject
			if err := h.Select(&objs, "SELECT * FROM lfs_objects WHERE repo_id = ?", r); err != nil {
				return err
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
	Rollback: func(ctx context.Context, h db.Handler) error {
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
