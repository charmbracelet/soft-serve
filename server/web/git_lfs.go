package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path"
	"path/filepath"
	"strconv"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/db"
	"github.com/charmbracelet/soft-serve/server/lfs"
	"github.com/charmbracelet/soft-serve/server/storage"
	"github.com/charmbracelet/soft-serve/server/store"
	"goji.io/pat"
)

// serviceLfsBatch handles a Git LFS batch requests.
// https://github.com/git-lfs/git-lfs/blob/main/docs/api/batch.md
// TODO: support refname & authentication
// POST: /<repo>.git/info/lfs/objects/batch
func serviceLfsBatch(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != lfs.MediaType {
		renderNotAcceptable(w)
		return
	}

	var batchRequest lfs.BatchRequest
	ctx := r.Context()
	logger := log.FromContext(ctx).WithPrefix("http.lfs")

	defer r.Body.Close() // nolint: errcheck
	if err := json.NewDecoder(r.Body).Decode(&batchRequest); err != nil {
		logger.Errorf("error decoding json: %s", err)
		return
	}

	// We only accept basic transfers for now
	// Default to basic if no transfer is specified
	if len(batchRequest.Transfers) > 0 {
		var isBasic bool
		for _, t := range batchRequest.Transfers {
			if t == lfs.TransferBasic {
				isBasic = true
				break
			}
		}

		if !isBasic {
			renderJSON(w, http.StatusUnprocessableEntity, lfs.ErrorResponse{
				Message: "unsupported transfer",
			})
			return
		}
	}

	be := backend.FromContext(ctx)
	name := pat.Param(r, "repo")
	repo, err := be.Repository(ctx, name)
	if err != nil {
		renderJSON(w, http.StatusNotFound, lfs.ErrorResponse{
			Message: "repository not found",
		})
		return
	}

	cfg := config.FromContext(ctx)
	dbx := db.FromContext(ctx)
	datastore := store.FromContext(ctx)
	// TODO: support S3 storage
	strg := storage.NewLocalStorage(filepath.Join(cfg.DataPath, "lfs"))

	baseHref := fmt.Sprintf("%s/%s/info/lfs/objects/basic", cfg.HTTP.PublicURL, name+".git")

	var batchResponse lfs.BatchResponse
	batchResponse.Transfer = lfs.TransferBasic
	batchResponse.HashAlgo = lfs.HashAlgorithmSHA256

	objects := make([]*lfs.ObjectResponse, 0, len(batchRequest.Objects))
	// XXX: We don't support objects TTL for now, probably implement that with
	// S3 using object "expires_at" & "expires_in"
	switch batchRequest.Operation {
	case lfs.OperationDownload:
		for _, o := range batchRequest.Objects {
			stat, err := strg.Stat(path.Join("objects", o.RelativePath()))
			if err != nil && !errors.Is(err, fs.ErrNotExist) {
				logger.Error("error getting object stat", "oid", o.Oid, "repo", name, "err", err)
				renderJSON(w, http.StatusInternalServerError, lfs.ErrorResponse{
					Message: "internal server error",
				})
				return
			}

			obj, err := datastore.GetLFSObjectByOid(ctx, dbx, repo.ID(), o.Oid)
			if err != nil && !errors.Is(err, db.ErrRecordNotFound) {
				logger.Error("error getting object from database", "oid", o.Oid, "repo", name, "err", err)
				renderJSON(w, http.StatusInternalServerError, lfs.ErrorResponse{
					Message: "internal server error",
				})
				return
			}

			if stat == nil {
				objects = append(objects, &lfs.ObjectResponse{
					Pointer: o,
					Error: &lfs.ObjectError{
						Code:    http.StatusNotFound,
						Message: "object not found",
					},
				})
			} else if stat.Size() != o.Size {
				objects = append(objects, &lfs.ObjectResponse{
					Pointer: o,
					Error: &lfs.ObjectError{
						Code:    http.StatusUnprocessableEntity,
						Message: "size mismatch",
					},
				})
			} else if o.IsValid() {
				objects = append(objects, &lfs.ObjectResponse{
					Pointer: o,
					Actions: map[string]*lfs.Link{
						lfs.ActionDownload: {
							Href: fmt.Sprintf("%s/%s", baseHref, o.Oid),
						},
					},
				})

				// If the object doesn't exist in the database, create it
				if stat != nil && obj.ID == 0 {
					if err := datastore.CreateLFSObject(ctx, dbx, repo.ID(), o.Oid, stat.Size()); err != nil {
						logger.Error("error creating object in datastore", "oid", o.Oid, "repo", name, "err", err)
						renderJSON(w, http.StatusInternalServerError, lfs.ErrorResponse{
							Message: "internal server error",
						})
						return
					}
				}
			} else {
				objects = append(objects, &lfs.ObjectResponse{
					Pointer: o,
					Error: &lfs.ObjectError{
						Code:    http.StatusUnprocessableEntity,
						Message: "invalid object",
					},
				})
			}
		}
	case lfs.OperationUpload:
		// Object upload logic happens in the "basic" API route
		for _, o := range batchRequest.Objects {
			if !o.IsValid() {
				objects = append(objects, &lfs.ObjectResponse{
					Pointer: o,
					Error: &lfs.ObjectError{
						Code:    http.StatusUnprocessableEntity,
						Message: "invalid object",
					},
				})
			} else {
				objects = append(objects, &lfs.ObjectResponse{
					Pointer: o,
					Actions: map[string]*lfs.Link{
						lfs.ActionUpload: {
							Href: fmt.Sprintf("%s/%s", baseHref, o.Oid),
						},
						// Verify uploaded objects
						// https://github.com/git-lfs/git-lfs/blob/main/docs/api/basic-transfers.md#verification
						lfs.ActionVerify: {
							Href: fmt.Sprintf("%s/verify", baseHref),
						},
					},
				})
			}
		}
	default:
		renderJSON(w, http.StatusUnprocessableEntity, lfs.ErrorResponse{
			Message: "unsupported operation",
		})
		return
	}

	batchResponse.Objects = objects
	renderJSON(w, http.StatusOK, batchResponse)
}

// serviceLfsBasic implements Git LFS basic transfer API
// https://github.com/git-lfs/git-lfs/blob/main/docs/api/basic-transfers.md
func serviceLfsBasic(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		serviceLfsBasicDownload(w, r)
	case http.MethodPut:
		serviceLfsBasicUpload(w, r)
	}
}

// GET: /<repo>.git/info/lfs/objects/basic/<oid>
func serviceLfsBasicDownload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	oid := pat.Param(r, "oid")
	cfg := config.FromContext(ctx)
	logger := log.FromContext(ctx).WithPrefix("http.lfs-basic")
	strg := storage.NewLocalStorage(filepath.Join(cfg.DataPath, "lfs"))

	obj, err := strg.Open(path.Join("objects", oid))
	if err != nil {
		logger.Error("error opening object", "oid", oid, "err", err)
		renderJSON(w, http.StatusNotFound, lfs.ErrorResponse{
			Message: "object not found",
		})
		return
	}

	stat, err := obj.Stat()
	if err != nil {
		logger.Error("error getting object stat", "oid", oid, "err", err)
		renderJSON(w, http.StatusInternalServerError, lfs.ErrorResponse{
			Message: "internal server error",
		})
		return
	}

	defer obj.Close() // nolint: errcheck
	if _, err := io.Copy(w, obj); err != nil {
		logger.Error("error copying object to response", "oid", oid, "err", err)
		renderJSON(w, http.StatusInternalServerError, lfs.ErrorResponse{
			Message: "internal server error",
		})
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(stat.Size(), 10))
	renderStatus(http.StatusOK)(w, nil)
}

// PUT: /<repo>.git/info/lfs/objects/basic/<oid>
func serviceLfsBasicUpload(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/octet-stream" {
		renderJSON(w, http.StatusUnsupportedMediaType, lfs.ErrorResponse{
			Message: "invalid content type",
		})
		return
	}

	ctx := r.Context()
	oid := pat.Param(r, "oid")
	cfg := config.FromContext(ctx)
	be := backend.FromContext(ctx)
	dbx := db.FromContext(ctx)
	datastore := store.FromContext(ctx)
	logger := log.FromContext(ctx).WithPrefix("http.lfs-basic")
	strg := storage.NewLocalStorage(filepath.Join(cfg.DataPath, "lfs"))
	name := pat.Param(r, "repo")

	defer r.Body.Close() // nolint: errcheck
	repo, err := be.Repository(ctx, name)
	if err != nil {
		renderJSON(w, http.StatusNotFound, lfs.ErrorResponse{
			Message: "repository not found",
		})
		return
	}

	// NOTE: Git LFS client will retry uploading the same object if there was a
	// partial error, so we need to skip existing objects.
	if _, err := datastore.GetLFSObjectByOid(ctx, dbx, repo.ID(), oid); err == nil {
		// Object exists, skip request
		io.Copy(io.Discard, r.Body) // nolint: errcheck
		renderStatus(http.StatusOK)(w, nil)
		return
	} else if !errors.Is(err, db.ErrRecordNotFound) {
		logger.Error("error getting object", "oid", oid, "err", err)
		renderJSON(w, http.StatusInternalServerError, lfs.ErrorResponse{
			Message: "internal server error",
		})
		return
	}

	if err := strg.Put(path.Join("objects", oid), r.Body); err != nil {
		logger.Error("error writing object", "oid", oid, "err", err)
		renderJSON(w, http.StatusInternalServerError, lfs.ErrorResponse{
			Message: "internal server error",
		})
		return
	}

	size, err := strconv.ParseInt(r.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		logger.Error("error parsing content length", "err", err)
		renderJSON(w, http.StatusBadRequest, lfs.ErrorResponse{
			Message: "invalid content length",
		})
		return
	}

	if err := datastore.CreateLFSObject(ctx, dbx, repo.ID(), oid, size); err != nil {
		logger.Error("error creating object", "oid", oid, "err", err)
		renderJSON(w, http.StatusInternalServerError, lfs.ErrorResponse{
			Message: "internal server error",
		})
		return
	}

	renderStatus(http.StatusOK)(w, nil)
}

// POST: /<repo>.git/info/lfs/objects/basic/verify
func serviceLfsBasicVerify(w http.ResponseWriter, r *http.Request) {
	var pointer lfs.Pointer
	ctx := r.Context()
	logger := log.FromContext(ctx).WithPrefix("http.lfs-basic")
	be := backend.FromContext(ctx)
	name := pat.Param(r, "repo")
	repo, err := be.Repository(ctx, name)
	if err != nil {
		renderJSON(w, http.StatusNotFound, lfs.ErrorResponse{
			Message: "repository not found",
		})
		return
	}

	defer r.Body.Close() // nolint: errcheck
	if err := json.NewDecoder(r.Body).Decode(&pointer); err != nil {
		logger.Error("error decoding json", "err", err)
		renderJSON(w, http.StatusBadRequest, lfs.ErrorResponse{
			Message: "invalid json",
		})
		return
	}

	cfg := config.FromContext(ctx)
	dbx := db.FromContext(ctx)
	datastore := store.FromContext(ctx)
	strg := storage.NewLocalStorage(filepath.Join(cfg.DataPath, "lfs"))
	if stat, err := strg.Stat(path.Join("objects", pointer.Oid)); err == nil {
		// Verify object is in the database.
		if _, err := datastore.GetLFSObjectByOid(ctx, dbx, repo.ID(), pointer.Oid); err != nil {
			if errors.Is(err, db.ErrRecordNotFound) {
				// Create missing object.
				if err := datastore.CreateLFSObject(ctx, dbx, repo.ID(), pointer.Oid, stat.Size()); err != nil {
					logger.Error("error creating object", "oid", pointer.Oid, "err", err)
					renderJSON(w, http.StatusInternalServerError, lfs.ErrorResponse{
						Message: "internal server error",
					})
					return
				}
			} else {
				logger.Error("error getting object", "oid", pointer.Oid, "err", err)
				renderJSON(w, http.StatusInternalServerError, lfs.ErrorResponse{
					Message: "internal server error",
				})
				return
			}
		}

		if pointer.IsValid() && stat.Size() == pointer.Size {
			renderStatus(http.StatusOK)(w, nil)
			return
		}
	} else if errors.Is(err, fs.ErrNotExist) {
		renderJSON(w, http.StatusNotFound, lfs.ErrorResponse{
			Message: "object not found",
		})
		return
	} else {
		logger.Error("error getting object", "oid", pointer.Oid, "err", err)
		renderJSON(w, http.StatusInternalServerError, lfs.ErrorResponse{
			Message: "internal server error",
		})
		return
	}
}

// POST: /<repo>.git/info/lfs/objects/locks
func serviceLfsLocksCreate(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != lfs.MediaType {
		renderNotAcceptable(w)
		return
	}

	panic("not implemented")
}

// renderJSON renders a JSON response with the given status code and value. It
// also sets the Content-Type header to the JSON LFS media type (application/vnd.git-lfs+json).
func renderJSON(w http.ResponseWriter, statusCode int, v interface{}) {
	w.Header().Set("Content-Type", lfs.MediaType)
	renderStatus(statusCode)(w, nil)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Error("error encoding json", "err", err)
	}
}

func renderNotAcceptable(w http.ResponseWriter) {
	renderStatus(http.StatusNotAcceptable)(w, nil)
}

func hdrLfs(w http.ResponseWriter) {
	w.Header().Set("Content-Type", lfs.MediaType)
	w.Header().Set("Accept", lfs.MediaType)
}
