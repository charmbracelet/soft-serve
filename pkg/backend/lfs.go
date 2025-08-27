package backend

import (
	"context"
	"errors"
	"io"
	"path"
	"path/filepath"
	"strconv"

	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/lfs"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/storage"
	"github.com/charmbracelet/soft-serve/pkg/store"
)

// StoreRepoMissingLFSObjects stores missing LFS objects for a repository.
func StoreRepoMissingLFSObjects(ctx context.Context, repo proto.Repository, dbx *db.DB, store store.Store, lfsClient lfs.Client) error {
	cfg := config.FromContext(ctx)
	repoID := strconv.FormatInt(repo.ID(), 10)
	lfsRoot := filepath.Join(cfg.DataPath, "lfs", repoID)

	// TODO: support S3 storage
	strg := storage.NewLocalStorage(lfsRoot)
	pointerChan := make(chan lfs.PointerBlob)
	errChan := make(chan error, 1)
	r, err := repo.Open()
	if err != nil {
		return err
	}

	go lfs.SearchPointerBlobs(ctx, r, pointerChan, errChan)

	download := func(pointers []lfs.Pointer) error {
		return lfsClient.Download(ctx, pointers, func(p lfs.Pointer, content io.ReadCloser, objectError error) error {
			if objectError != nil {
				return objectError
			}

			defer content.Close() 
			return dbx.TransactionContext(ctx, func(tx *db.Tx) error {
				if err := store.CreateLFSObject(ctx, tx, repo.ID(), p.Oid, p.Size); err != nil {
					return db.WrapError(err)
				}

				_, err := strg.Put(path.Join("objects", p.RelativePath()), content)
				return err
			})
		})
	}

	var batch []lfs.Pointer
	for pointer := range pointerChan {
		obj, err := store.GetLFSObjectByOid(ctx, dbx, repo.ID(), pointer.Oid)
		if err != nil && !errors.Is(err, db.ErrRecordNotFound) {
			return db.WrapError(err)
		}

		exist, err := strg.Exists(path.Join("objects", pointer.RelativePath()))
		if err != nil {
			return err
		}

		if exist && obj.ID == 0 {
			if err := store.CreateLFSObject(ctx, dbx, repo.ID(), pointer.Oid, pointer.Size); err != nil {
				return db.WrapError(err)
			}
		} else {
			batch = append(batch, pointer.Pointer)
			// Limit batch requests to 20 objects
			if len(batch) >= 20 {
				if err := download(batch); err != nil {
					return err
				}

				batch = nil
			}
		}
	}

	if err, ok := <-errChan; ok {
		return err
	}

	return nil
}
