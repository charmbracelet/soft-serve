package backend

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/lfs"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/storage"
	"github.com/charmbracelet/soft-serve/pkg/store"
	"golang.org/x/sync/errgroup"
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

			defer content.Close() //nolint: errcheck
			// Disk-ahead-of-DB invariant: strg.Put writes the object to the
			// filesystem; CreateLFSObject writes the DB row. If strg.Put succeeds
			// but the DB transaction rolls back, the object file remains on disk
			// without a corresponding DB row. The re-registration path below
			// (obj != nil && obj.ID == 0) handles this case on the next download
			// attempt by re-inserting the DB row without re-downloading the data.
			// Write object to disk first (outside transaction to avoid holding
			// DB slot for long downloads). If DB insert fails, delete file.
			objPath := path.Join("objects", p.RelativePath())
			if _, err := strg.Put(objPath, content); err != nil {
				return fmt.Errorf("failed to write LFS object to disk: %w", err)
			}
			return dbx.TransactionContext(ctx, func(tx *db.Tx) error {
				if err := store.CreateLFSObject(ctx, tx, repo.ID(), p.Oid, p.Size); err != nil {
					// DB insert failed — clean up on-disk file to avoid orphan.
					// Use strg.Delete so the path is resolved relative to lfsRoot,
					// not the process working directory.
					if rmErr := strg.Delete(objPath); rmErr != nil {
						return errors.Join(err, rmErr)
					}
					return err // strg.Delete succeeded, propagate DB error
				}
				return nil
			})
		})
	}

	const lfsBatchSize = 20
	var batch []lfs.Pointer
	var lookupOids []string
	var pointerBlobs []lfs.PointerBlob
	var objMap = make(map[string]*models.LFSObject)

	// Drain the channel once, collecting both OIDs and blobs.
	// A second range over a closed channel would iterate zero times.
	for pointer := range pointerChan {
		lookupOids = append(lookupOids, pointer.Oid)
		pointerBlobs = append(pointerBlobs, pointer)
	}

	// Batch fetch all objects to eliminate N+1 query
	objects, err := store.GetLFSObjectsByOids(ctx, dbx, repo.ID(), lookupOids)
	if err != nil {
		return db.WrapError(err)
	}
	for _, obj := range objects {
		objMap[obj.Oid] = &obj
	}

	// Pre-fetch filesystem existence for all pointer blobs concurrently.
	existsByOid := make(map[string]bool, len(pointerBlobs))
	var existsMu sync.Mutex
	existsGroup, _ := errgroup.WithContext(ctx)
	for _, pointer := range pointerBlobs {
		pointer := pointer
		existsGroup.Go(func() error {
			exist, err := strg.Exists(path.Join("objects", pointer.RelativePath()))
			if err != nil {
				return err
			}
			existsMu.Lock()
			existsByOid[pointer.Oid] = exist
			existsMu.Unlock()
			return nil
		})
	}
	if err := existsGroup.Wait(); err != nil {
		return err
	}

	for _, pointer := range pointerBlobs {
		obj, exists := objMap[pointer.Oid]
		if !exists {
			// Object not found in DB — skip (shouldn't happen if scanner worked correctly)
			continue
		}

		exist := existsByOid[pointer.Oid]

		if exist && obj.ID != 0 {
			// fully synced — skip
			continue
		}
		if exist && obj.ID == 0 {
			// Disk-ahead-of-DB recovery: object is on disk but not in the DB.
			// Validate the pointer before re-registering it to guard against
			// malformed OIDs reaching the DB (defense-in-depth; the LFS scanner
			// already validates OIDs, but we check again here to be explicit).
			if !pointer.IsValid() {
				return fmt.Errorf("lfs: invalid pointer during re-registration: oid=%s", pointer.Oid)
			}
			if err := store.CreateLFSObject(ctx, dbx, repo.ID(), pointer.Oid, pointer.Size); err != nil {
				return db.WrapError(err)
			}
			continue
		}
		// not on disk — add to download batch
		batch = append(batch, pointer.Pointer)
		// Limit batch requests to lfsBatchSize objects
		if len(batch) >= lfsBatchSize {
			if err := download(batch); err != nil {
				return err
			}

			batch = nil
		}
	}

	if len(batch) > 0 {
		if err := download(batch); err != nil {
			return err
		}
	}

	// errChan is closed by SearchPointerBlobs after wg.Wait() completes.
	// If SearchPointerBlobs sent an error before closing, ok is true and err
	// holds the error. If it closed without sending (no error), ok is false
	// and err is nil — the zero value — which we correctly ignore.
	if err, ok := <-errChan; ok {
		return err
	}

	return nil
}
