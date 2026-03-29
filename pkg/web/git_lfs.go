package web

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"charm.land/log/v2"
	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/lfs"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/storage"
	"github.com/charmbracelet/soft-serve/pkg/store"
	"github.com/gorilla/mux"
)

// lfsOidPattern matches a valid Git LFS bare object ID (64 lowercase hex chars).
// lfs.Pointer.Oid stores the bare hex internally — the "sha256:" prefix present
// in pointer files and batch JSON is stripped on parse. Callers must NOT prepend
// the prefix before matching; the route-level pattern for download also uses bare
// hex (`[0-9a-f]{64}`).
var lfsOidPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// serviceLfsBatch handles a Git LFS batch requests.
// https://github.com/git-lfs/git-lfs/blob/main/docs/api/batch.md
// TODO: support refname
// POST: /<repo>.git/info/lfs/objects/batch
func serviceLfsBatch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithPrefix("http.lfs")

	if !isLfs(r) {
		logger.Errorf("invalid content type: %s", r.Header.Get("Content-Type"))
		renderNotAcceptable(w)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var batchRequest lfs.BatchRequest
	defer r.Body.Close() //nolint: errcheck
	if err := json.NewDecoder(r.Body).Decode(&batchRequest); err != nil {
		logger.Errorf("error decoding json: %s", err)
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			renderJSON(w, r, http.StatusRequestEntityTooLarge, lfs.ErrorResponse{
				Message: "request body too large",
			})
		} else {
			renderJSON(w, r, http.StatusUnprocessableEntity, lfs.ErrorResponse{
				Message: "invalid request body",
			})
		}
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
			renderJSON(w, r, http.StatusUnprocessableEntity, lfs.ErrorResponse{
				Message: "unsupported transfer",
			})
			return
		}
	}

	if len(batchRequest.Objects) == 0 {
		renderJSON(w, r, http.StatusUnprocessableEntity, lfs.ErrorResponse{
			Message: "no objects found",
		})
		return
	}

	// Cap the number of objects per batch request to prevent a single
	// request from triggering thousands of sequential DB + FS round-trips.
	// 1 000 matches the GitHub LFS limit and is sufficient for real clients.
	const maxBatchObjects = 1000
	if len(batchRequest.Objects) > maxBatchObjects {
		renderJSON(w, r, http.StatusUnprocessableEntity, lfs.ErrorResponse{
			Message: fmt.Sprintf("batch request exceeds maximum object count of %d", maxBatchObjects),
		})
		return
	}

	name := mux.Vars(r)["repo"]
	repo := proto.RepositoryFromContext(ctx)
	if repo == nil {
		renderJSON(w, r, http.StatusNotFound, lfs.ErrorResponse{
			Message: "repository not found",
		})
		return
	}

	cfg := config.FromContext(ctx)
	dbx := db.FromContext(ctx)
	datastore := store.FromContext(ctx)
	// TODO: support S3 storage
	repoID := strconv.FormatInt(repo.ID(), 10)
	strg := storage.NewLocalStorage(filepath.Join(cfg.DataPath, "lfs", repoID))

	baseHref := fmt.Sprintf("%s/%s/info/lfs/objects/basic", cfg.HTTP.PublicURL, name+".git")

	var batchResponse lfs.BatchResponse
	batchResponse.Transfer = lfs.TransferBasic
	batchResponse.HashAlgo = lfs.HashAlgorithmSHA256

	objects := make([]*lfs.ObjectResponse, 0, len(batchRequest.Objects))
	// XXX: We don't support objects TTL for now, probably implement that with
	// S3 using object "expires_at" & "expires_in"
	switch batchRequest.Operation {
	case lfs.OperationDownload:
		// Bulk-fetch all requested OIDs in a single query to avoid N+1 DB
		// round-trips when a batch contains many objects.
		oids := make([]string, len(batchRequest.Objects))
		for i, o := range batchRequest.Objects {
			oids[i] = o.Oid
		}
		dbObjs, err := datastore.GetLFSObjectsByOids(ctx, dbx, repo.ID(), oids)
		if err != nil {
			logger.Error("error bulk-fetching LFS objects from database", "repo", name, "err", err)
			renderJSON(w, r, http.StatusInternalServerError, lfs.ErrorResponse{
				Message: "internal server error",
			})
			return
		}
		dbObjsByOid := make(map[string]models.LFSObject, len(dbObjs))
		for _, dbObj := range dbObjs {
			dbObjsByOid[dbObj.Oid] = dbObj
		}

		for _, o := range batchRequest.Objects {
			exist, err := strg.Exists(path.Join("objects", o.RelativePath()))
			if err != nil && !errors.Is(err, fs.ErrNotExist) {
				logger.Error("error getting object stat", "oid", o.Oid, "repo", name, "err", err)
				renderJSON(w, r, http.StatusInternalServerError, lfs.ErrorResponse{
					Message: "internal server error",
				})
				return
			}

			obj, objNotFound := dbObjsByOid[o.Oid]
			// objNotFound is true when the OID is absent from the DB map.
			objInDB := !objNotFound

			if !exist {
				objects = append(objects, &lfs.ObjectResponse{
					Pointer: o,
					Error: &lfs.ObjectError{
						Code:    http.StatusNotFound,
						Message: "object not found",
					},
				})
			} else if objInDB && obj.Size != o.Size {
				objects = append(objects, &lfs.ObjectResponse{
					Pointer: o,
					Error: &lfs.ObjectError{
						Code:    http.StatusUnprocessableEntity,
						Message: "size mismatch",
					},
				})
			} else if o.IsValid() {
				download := &lfs.Link{
					Href: fmt.Sprintf("%s/%s", baseHref, o.Oid),
				}

				objects = append(objects, &lfs.ObjectResponse{
					Pointer: o,
					Actions: map[string]*lfs.Link{
						lfs.ActionDownload: download,
					},
				})

				// If the object exists on disk but not in the database,
				// re-register it (disk-ahead-of-DB recovery path).
				if exist && !objInDB {
					if err := datastore.CreateLFSObject(ctx, dbx, repo.ID(), o.Oid, o.Size); err != nil {
						logger.Error("error creating object in datastore", "oid", o.Oid, "repo", name, "err", err)
						renderJSON(w, r, http.StatusInternalServerError, lfs.ErrorResponse{
							Message: "internal server error",
						})
						return
					}
				}
			} else {
				logger.Error("invalid object", "oid", o.Oid, "repo", name)
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
		// Check authorization
		accessLevel := access.FromContext(ctx)
		if accessLevel < access.ReadWriteAccess {
			askCredentials(w, r)
			renderJSON(w, r, http.StatusForbidden, lfs.ErrorResponse{
				Message: "write access required",
			})
			return
		}

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
				upload := &lfs.Link{
					Href: fmt.Sprintf("%s/%s", baseHref, o.Oid),
					Header: map[string]string{
						// NOTE: git-lfs v2.5.0 sets the Content-Type based on the uploaded file.
						// This ensures that the client always uses the designated value for the header.
						"Content-Type": "application/octet-stream",
					},
				}
				verify := &lfs.Link{
					Href: fmt.Sprintf("%s/verify", baseHref),
				}

				objects = append(objects, &lfs.ObjectResponse{
					Pointer: o,
					Actions: map[string]*lfs.Link{
						lfs.ActionUpload: upload,
						// Verify uploaded objects
						// https://github.com/git-lfs/git-lfs/blob/main/docs/api/basic-transfers.md#verification
						lfs.ActionVerify: verify,
					},
				})
			}
		}
	default:
		renderJSON(w, r, http.StatusUnprocessableEntity, lfs.ErrorResponse{
			Message: "unsupported operation",
		})
		return
	}

	batchResponse.Objects = objects
	w.Header().Set("Cache-Control", "no-store, private")
	renderJSON(w, r, http.StatusOK, batchResponse)
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
	oid := mux.Vars(r)["oid"]
	repo := proto.RepositoryFromContext(ctx)
	if repo == nil {
		renderStatus(http.StatusNotFound)(w, r)
		return
	}
	cfg := config.FromContext(ctx)
	logger := log.FromContext(ctx).WithPrefix("http.lfs-basic")
	datastore := store.FromContext(ctx)
	dbx := db.FromContext(ctx)
	repoID := strconv.FormatInt(repo.ID(), 10)
	strg := storage.NewLocalStorage(filepath.Join(cfg.DataPath, "lfs", repoID))

	obj, err := datastore.GetLFSObjectByOid(ctx, dbx, repo.ID(), oid)
	if err != nil && !errors.Is(err, db.ErrRecordNotFound) {
		logger.Error("error getting object from database", "oid", oid, "repo", repo.Name(), "err", err)
		renderJSON(w, r, http.StatusInternalServerError, lfs.ErrorResponse{
			Message: "internal server error",
		})
		return
	}

	// Validate OID explicitly even though the route regex already constrains it,
	// to guard against future route-regex relaxation.
	if !lfsOidPattern.MatchString(oid) {
		renderJSON(w, r, http.StatusUnprocessableEntity, lfs.ErrorResponse{Message: "invalid oid format"})
		return
	}
	pointer := lfs.Pointer{Oid: oid}
	f, err := strg.Open(path.Join("objects", pointer.RelativePath()))
	if err != nil {
		logger.Error("error opening object", "oid", oid, "err", err)
		renderJSON(w, r, http.StatusNotFound, lfs.ErrorResponse{
			Message: "object not found",
		})
		return
	}

	var size int64
	if obj.ID != 0 {
		size = obj.Size
	} else {
		if stat, err := strg.Stat(path.Join("objects", pointer.RelativePath())); err == nil {
			size = stat.Size()
			// Object exists on disk but is not in the database; register it now.
			if err := datastore.CreateLFSObject(ctx, dbx, repo.ID(), oid, size); err != nil {
				logger.Error("error creating lfs object in datastore", "oid", oid, "repo", repo.Name(), "err", err)
				renderJSON(w, r, http.StatusInternalServerError, lfs.ErrorResponse{
					Message: "internal server error",
				})
				return
			}
		}
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	if size > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	}
	defer f.Close() //nolint: errcheck
	// Once io.Copy begins writing the response body, status code and headers
	// are already flushed. Any error mid-stream cannot be surfaced as a new
	// HTTP status; the client will receive a truncated response.
	if _, err := io.Copy(w, f); err != nil {
		logger.Error("error copying object to response", "oid", oid, "err", err)
		return
	}
}

// PUT: /<repo>.git/info/lfs/objects/basic/<oid>
func serviceLfsBasicUpload(w http.ResponseWriter, r *http.Request) {
	const maxLFSObjectSize = 5 << 30 // 5 GiB
	r.Body = http.MaxBytesReader(w, r.Body, maxLFSObjectSize)

	if !isBinary(r) {
		renderJSON(w, r, http.StatusUnsupportedMediaType, lfs.ErrorResponse{
			Message: "invalid content type",
		})
		return
	}

	ctx := r.Context()
	oid := mux.Vars(r)["oid"]

	// Validate OID explicitly for defence-in-depth (route regex already
	// constrains it, but this guards against future route-regex relaxation).
	if !lfsOidPattern.MatchString(oid) {
		renderJSON(w, r, http.StatusUnprocessableEntity, lfs.ErrorResponse{Message: "invalid oid format"})
		return
	}

	cfg := config.FromContext(ctx)
	dbx := db.FromContext(ctx)
	datastore := store.FromContext(ctx)
	logger := log.FromContext(ctx).WithPrefix("http.lfs-basic")
	repo := proto.RepositoryFromContext(ctx)
	if repo == nil {
		renderStatus(http.StatusNotFound)(w, r)
		return
	}
	repoID := strconv.FormatInt(repo.ID(), 10)
	strg := storage.NewLocalStorage(filepath.Join(cfg.DataPath, "lfs", repoID))

	defer r.Body.Close() //nolint: errcheck

	// NOTE: Git LFS client will retry uploading the same object if there was a
	// partial error, so we need to skip existing objects. Do NOT drain the body
	// here — the client can send up to 5 GiB and discarding it wastes CPU and
	// bandwidth. Responding 200 immediately is correct; git-lfs treats a closed
	// connection on a PUT as a successful upload.
	if _, err := datastore.GetLFSObjectByOid(ctx, dbx, repo.ID(), oid); err == nil {
		// Object exists, skip request.
		renderStatus(http.StatusOK)(w, r)
		return
	} else if !errors.Is(err, db.ErrRecordNotFound) {
		logger.Error("error getting object", "oid", oid, "err", err)
		renderJSON(w, r, http.StatusInternalServerError, lfs.ErrorResponse{
			Message: "internal server error",
		})
		return
	}

	pointer := lfs.Pointer{Oid: oid}
	h := sha256.New()
	tee := io.TeeReader(r.Body, h)
	n, err := strg.Put(path.Join("objects", pointer.RelativePath()), tee)
	if err != nil {
		logger.Error("error writing object", "oid", oid, "err", err)
		renderJSON(w, r, http.StatusInternalServerError, lfs.ErrorResponse{
			Message: "internal server error",
		})
		return
	}

	actualOID := hex.EncodeToString(h.Sum(nil))
	if actualOID != oid {
		// Content does not match the declared OID; remove the corrupt file.
		if delErr := strg.Delete(path.Join("objects", pointer.RelativePath())); delErr != nil {
			logger.Error("error deleting mismatched object", "oid", oid, "err", delErr)
		}
		renderJSON(w, r, http.StatusUnprocessableEntity, lfs.ErrorResponse{
			Message: "object hash mismatch",
		})
		return
	}

	if err := datastore.CreateLFSObject(ctx, dbx, repo.ID(), oid, n); err != nil {
		if errors.Is(err, db.ErrDuplicateKey) {
			// A concurrent upload for the same OID already committed the record;
			// treat as idempotent success (git-lfs sends parallel PUTs for large pushes).
			renderStatus(http.StatusOK)(w, nil)
			return
		}
		logger.Error("error creating object", "oid", oid, "err", err)
		renderJSON(w, r, http.StatusInternalServerError, lfs.ErrorResponse{
			Message: "internal server error",
		})
		return
	}

	renderStatus(http.StatusOK)(w, nil)
}

// POST: /<repo>.git/info/lfs/objects/basic/verify
func serviceLfsBasicVerify(w http.ResponseWriter, r *http.Request) {
	if !isLfs(r) {
		renderNotAcceptable(w)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var pointer lfs.Pointer
	ctx := r.Context()
	logger := log.FromContext(ctx).WithPrefix("http.lfs-basic")
	repo := proto.RepositoryFromContext(ctx)
	if repo == nil {
		logger.Error("error getting repository from context")
		renderJSON(w, r, http.StatusNotFound, lfs.ErrorResponse{
			Message: "repository not found",
		})
		return
	}

	defer r.Body.Close() //nolint: errcheck
	if err := json.NewDecoder(r.Body).Decode(&pointer); err != nil {
		logger.Error("error decoding json", "err", err)
		renderJSON(w, r, http.StatusBadRequest, lfs.ErrorResponse{
			Message: "invalid request body",
		})
		return
	}

	if !lfsOidPattern.MatchString(pointer.Oid) {
		renderJSON(w, r, http.StatusUnprocessableEntity, lfs.ErrorResponse{Message: "invalid oid format"})
		return
	}

	cfg := config.FromContext(ctx)
	dbx := db.FromContext(ctx)
	datastore := store.FromContext(ctx)
	repoID := strconv.FormatInt(repo.ID(), 10)
	strg := storage.NewLocalStorage(filepath.Join(cfg.DataPath, "lfs", repoID))
	if stat, err := strg.Stat(path.Join("objects", pointer.RelativePath())); err == nil {
		// Verify object is in the database.
		obj, err := datastore.GetLFSObjectByOid(ctx, dbx, repo.ID(), pointer.Oid)
		if err != nil {
			if errors.Is(err, db.ErrRecordNotFound) {
				logger.Error("object not found", "oid", pointer.Oid)
				renderJSON(w, r, http.StatusNotFound, lfs.ErrorResponse{
					Message: "object not found",
				})
				return
			}
			logger.Error("error getting object", "oid", pointer.Oid, "err", err)
			renderJSON(w, r, http.StatusInternalServerError, lfs.ErrorResponse{
				Message: "internal server error",
			})
			return
		}

		if obj.Size != pointer.Size {
			renderJSON(w, r, http.StatusBadRequest, lfs.ErrorResponse{
				Message: "object size mismatch",
			})
			return
		}

		if pointer.IsValid() && stat.Size() == pointer.Size {
			w.Header().Set("Content-Type", "application/vnd.git-lfs+json")
			renderJSON(w, r, http.StatusOK, map[string]string{"message": "verified"})
			return
		}
		renderJSON(w, r, http.StatusUnprocessableEntity, lfs.ErrorResponse{Message: "size mismatch"})
		return
	} else if errors.Is(err, fs.ErrNotExist) {
		logger.Error("file not found", "oid", pointer.Oid)
		renderJSON(w, r, http.StatusNotFound, lfs.ErrorResponse{
			Message: "object not found",
		})
		return
	} else {
		logger.Error("error getting object", "oid", pointer.Oid, "err", err)
		renderJSON(w, r, http.StatusInternalServerError, lfs.ErrorResponse{
			Message: "internal server error",
		})
		return
	}
}

func serviceLfsLocks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		serviceLfsLocksGet(w, r)
	case http.MethodPost:
		serviceLfsLocksCreate(w, r)
	default:
		renderMethodNotAllowed(w, r)
	}
}

// POST: /<repo>.git/info/lfs/objects/locks
func serviceLfsLocksCreate(w http.ResponseWriter, r *http.Request) {
	if !isLfs(r) {
		renderNotAcceptable(w)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	ctx := r.Context()
	logger := log.FromContext(ctx).WithPrefix("http.lfs-locks")

	var req lfs.LockCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("error decoding json", "err", err)
		renderJSON(w, r, http.StatusBadRequest, lfs.ErrorResponse{
			Message: "invalid request body",
		})
		return
	}

	repo := proto.RepositoryFromContext(ctx)
	if repo == nil {
		logger.Error("error getting repository from context")
		renderJSON(w, r, http.StatusNotFound, lfs.ErrorResponse{
			Message: "repository not found",
		})
		return
	}

	user := proto.UserFromContext(ctx)
	if user == nil {
		logger.Error("error getting user from context")
		renderJSON(w, r, http.StatusNotFound, lfs.ErrorResponse{
			Message: "user not found",
		})
		return
	}

	dbx := db.FromContext(ctx)
	datastore := store.FromContext(ctx)
	if err := datastore.CreateLFSLockForUser(ctx, dbx, repo.ID(), user.ID(), req.Path, req.Ref.Name); err != nil {
		err = db.WrapError(err)
		if errors.Is(err, db.ErrDuplicateKey) {
			errResp := lfs.LockResponse{
				ErrorResponse: lfs.ErrorResponse{
					Message: "lock already exists",
				},
			}
			lock, err := datastore.GetLFSLockForUserPath(ctx, dbx, repo.ID(), user.ID(), req.Path)
			if err == nil {
				errResp.Lock = lfs.Lock{
					ID:       strconv.FormatInt(lock.ID, 10),
					Path:     lock.Path,
					LockedAt: lock.CreatedAt,
				}
				lockOwner := lfs.Owner{
					Name: user.Username(),
				}
				if lock.UserID != user.ID() {
					owner, err := datastore.GetUserByID(ctx, dbx, lock.UserID)
					if err != nil {
						logger.Error("error getting lock owner", "err", err)
						renderJSON(w, r, http.StatusInternalServerError, lfs.ErrorResponse{
							Message: "internal server error",
						})
						return
					}
					lockOwner.Name = owner.Username
				}
				errResp.Lock.Owner = lockOwner
			}
			renderJSON(w, r, http.StatusConflict, errResp)
			return
		}
		logger.Error("error creating lock", "err", err)
		renderJSON(w, r, http.StatusInternalServerError, lfs.ErrorResponse{
			Message: "internal server error",
		})
		return
	}

	lock, err := datastore.GetLFSLockForUserPath(ctx, dbx, repo.ID(), user.ID(), req.Path)
	if err != nil {
		logger.Error("error getting lock", "err", err)
		renderJSON(w, r, http.StatusInternalServerError, lfs.ErrorResponse{
			Message: "internal server error",
		})
		return
	}

	renderJSON(w, r, http.StatusCreated, lfs.LockResponse{
		Lock: lfs.Lock{
			ID:       strconv.FormatInt(lock.ID, 10),
			Path:     lock.Path,
			LockedAt: lock.CreatedAt,
			Owner: lfs.Owner{
				Name: user.Username(),
			},
		},
	})
}

// GET: /<repo>.git/info/lfs/objects/locks
func serviceLfsLocksGet(w http.ResponseWriter, r *http.Request) {
	accept := r.Header.Get("Accept")
	if !strings.HasPrefix(accept, lfs.MediaType) {
		renderNotAcceptable(w)
		return
	}

	parseLocksQuery := func(values url.Values) (path string, id int64, cursor int, limit int, refspec string) {
		path = values.Get("path")
		idStr := values.Get("id")
		if idStr != "" {
			id, _ = strconv.ParseInt(idStr, 10, 64)
		}
		cursorStr := values.Get("cursor")
		if cursorStr != "" {
			cursor, _ = strconv.Atoi(cursorStr)
		}
		limitStr := values.Get("limit")
		if limitStr != "" {
			limit, _ = strconv.Atoi(limitStr)
		}
		refspec = values.Get("refspec")
		return
	}

	ctx := r.Context()
	// TODO: respect refspec
	path, id, cursor, limit, _ := parseLocksQuery(r.URL.Query())
	if limit > 100 {
		limit = 100
	} else if limit <= 0 {
		limit = lfs.DefaultLocksLimit
	}

	// cursor is the page number
	if cursor <= 0 {
		cursor = 1
	}
	const maxCursorPage = 10000
	if cursor > maxCursorPage {
		cursor = maxCursorPage
	}

	logger := log.FromContext(ctx).WithPrefix("http.lfs-locks")
	dbx := db.FromContext(ctx)
	datastore := store.FromContext(ctx)
	repo := proto.RepositoryFromContext(ctx)
	if repo == nil {
		logger.Error("error getting repository from context")
		renderJSON(w, r, http.StatusNotFound, lfs.ErrorResponse{
			Message: "repository not found",
		})
		return
	}

	if id > 0 {
		lock, err := datastore.GetLFSLockByID(ctx, dbx, id)
		if err != nil {
			if errors.Is(err, db.ErrRecordNotFound) {
				renderJSON(w, r, http.StatusNotFound, lfs.ErrorResponse{
					Message: "lock not found",
				})
				return
			}
			logger.Error("error getting lock", "err", err)
			renderJSON(w, r, http.StatusInternalServerError, lfs.ErrorResponse{
				Message: "internal server error",
			})
			return
		}

		// Scope check: reject locks belonging to other repos.
		if lock.RepoID != repo.ID() {
			renderJSON(w, r, http.StatusNotFound, lfs.ErrorResponse{
				Message: "lock not found",
			})
			return
		}

		owner, err := datastore.GetUserByID(ctx, dbx, lock.UserID)
		if err != nil {
			logger.Error("error getting lock owner", "err", err)
			renderJSON(w, r, http.StatusInternalServerError, lfs.ErrorResponse{
				Message: "internal server error",
			})
			return
		}

		renderJSON(w, r, http.StatusOK, lfs.LockListResponse{
			Locks: []lfs.Lock{
				{
					ID:       strconv.FormatInt(lock.ID, 10),
					Path:     lock.Path,
					LockedAt: lock.CreatedAt,
					Owner: lfs.Owner{
						Name: owner.Username,
					},
				},
			},
		})
		return
	} else if path != "" {
		lock, err := datastore.GetLFSLockForPath(ctx, dbx, repo.ID(), path)
		if err != nil {
			if errors.Is(err, db.ErrRecordNotFound) {
				renderJSON(w, r, http.StatusNotFound, lfs.ErrorResponse{
					Message: "lock not found",
				})
				return
			}
			logger.Error("error getting lock", "err", err)
			renderJSON(w, r, http.StatusInternalServerError, lfs.ErrorResponse{
				Message: "internal server error",
			})
			return
		}

		owner, err := datastore.GetUserByID(ctx, dbx, lock.UserID)
		if err != nil {
			logger.Error("error getting lock owner", "err", err)
			renderJSON(w, r, http.StatusInternalServerError, lfs.ErrorResponse{
				Message: "internal server error",
			})
			return
		}

		renderJSON(w, r, http.StatusOK, lfs.LockListResponse{
			Locks: []lfs.Lock{
				{
					ID:       strconv.FormatInt(lock.ID, 10),
					Path:     lock.Path,
					LockedAt: lock.CreatedAt,
					Owner: lfs.Owner{
						Name: owner.Username,
					},
				},
			},
		})
		return
	}

	locks, err := datastore.GetLFSLocks(ctx, dbx, repo.ID(), cursor, limit)
	if err != nil {
		logger.Error("error getting locks", "err", err)
		renderJSON(w, r, http.StatusInternalServerError, lfs.ErrorResponse{
			Message: "internal server error",
		})
		return
	}

	lockList := make([]lfs.Lock, len(locks))
	users := map[int64]models.User{}
	for i, lock := range locks {
		owner, ok := users[lock.UserID]
		if !ok {
			owner, err = datastore.GetUserByID(ctx, dbx, lock.UserID)
			if err != nil {
				logger.Error("error getting lock owner", "err", err)
				renderJSON(w, r, http.StatusInternalServerError, lfs.ErrorResponse{
					Message: "internal server error",
				})
				return
			}
			users[lock.UserID] = owner
		}

		lockList[i] = lfs.Lock{
			ID:       strconv.FormatInt(lock.ID, 10),
			Path:     lock.Path,
			LockedAt: lock.CreatedAt,
			Owner: lfs.Owner{
				Name: owner.Username,
			},
		}
	}

	resp := lfs.LockListResponse{
		Locks: lockList,
	}
	if len(locks) == limit {
		resp.NextCursor = strconv.Itoa(cursor + 1)
	}

	renderJSON(w, r, http.StatusOK, resp)
}

// POST: /<repo>.git/info/lfs/objects/locks/verify
func serviceLfsLocksVerify(w http.ResponseWriter, r *http.Request) {
	if !isLfs(r) {
		renderNotAcceptable(w)
		return
	}

	ctx := r.Context()
	logger := log.FromContext(ctx).WithPrefix("http.lfs-locks")
	repo := proto.RepositoryFromContext(ctx)
	if repo == nil {
		logger.Error("error getting repository from context")
		renderJSON(w, r, http.StatusNotFound, lfs.ErrorResponse{
			Message: "repository not found",
		})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req lfs.LockVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("error decoding request", "err", err)
		renderJSON(w, r, http.StatusBadRequest, lfs.ErrorResponse{
			Message: "invalid request body",
		})
		return
	}

	// TODO: refspec
	cursor, _ := strconv.Atoi(req.Cursor)
	if cursor <= 0 {
		cursor = 1
	}

	limit := req.Limit
	if limit > 100 {
		limit = 100
	} else if limit <= 0 {
		limit = lfs.DefaultLocksLimit
	}

	dbx := db.FromContext(ctx)
	datastore := store.FromContext(ctx)
	user := proto.UserFromContext(ctx)
	ours := make([]lfs.Lock, 0)
	theirs := make([]lfs.Lock, 0)

	var resp lfs.LockVerifyResponse
	locks, err := datastore.GetLFSLocks(ctx, dbx, repo.ID(), cursor, limit)
	if err != nil {
		logger.Error("error getting locks", "err", err)
		renderJSON(w, r, http.StatusInternalServerError, lfs.ErrorResponse{
			Message: "internal server error",
		})
		return
	}

	users := map[int64]models.User{}
	for _, lock := range locks {
		owner, ok := users[lock.UserID]
		if !ok {
			owner, err = datastore.GetUserByID(ctx, dbx, lock.UserID)
			if err != nil {
				logger.Error("error getting lock owner", "err", err)
				renderJSON(w, r, http.StatusInternalServerError, lfs.ErrorResponse{
					Message: "internal server error",
				})
				return
			}
			users[lock.UserID] = owner
		}

		l := lfs.Lock{
			ID:       strconv.FormatInt(lock.ID, 10),
			Path:     lock.Path,
			LockedAt: lock.CreatedAt,
			Owner: lfs.Owner{
				Name: owner.Username,
			},
		}

		if user != nil && user.ID() == lock.UserID {
			ours = append(ours, l)
		} else {
			theirs = append(theirs, l)
		}
	}

	resp.Ours = ours
	resp.Theirs = theirs

	if len(locks) == limit {
		resp.NextCursor = strconv.Itoa(cursor + 1)
	}

	renderJSON(w, r, http.StatusOK, resp)
}

// POST: /<repo>.git/info/lfs/objects/locks/:lockID/unlock
func serviceLfsLocksDelete(w http.ResponseWriter, r *http.Request) {
	if !isLfs(r) {
		renderNotAcceptable(w)
		return
	}

	ctx := r.Context()
	logger := log.FromContext(ctx).WithPrefix("http.lfs-locks")
	lockIDStr := mux.Vars(r)["lock_id"]
	if lockIDStr == "" {
		logger.Error("error getting lock id")
		renderJSON(w, r, http.StatusBadRequest, lfs.ErrorResponse{
			Message: "invalid request",
		})
		return
	}

	lockID, err := strconv.ParseInt(lockIDStr, 10, 64)
	if err != nil {
		logger.Error("error parsing lock id", "err", err)
		renderJSON(w, r, http.StatusBadRequest, lfs.ErrorResponse{
			Message: "invalid request",
		})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req lfs.LockDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("error decoding request", "err", err)
		renderJSON(w, r, http.StatusBadRequest, lfs.ErrorResponse{
			Message: "invalid request body",
		})
		return
	}

	dbx := db.FromContext(ctx)
	datastore := store.FromContext(ctx)
	repo := proto.RepositoryFromContext(ctx)
	if repo == nil {
		logger.Error("error getting repository from context")
		renderJSON(w, r, http.StatusNotFound, lfs.ErrorResponse{
			Message: "repository not found",
		})
		return
	}

	// The lock being deleted
	lock, err := datastore.GetLFSLockByID(ctx, dbx, lockID)
	if err != nil {
		logger.Error("error getting lock", "err", err)
		renderJSON(w, r, http.StatusNotFound, lfs.ErrorResponse{
			Message: "lock not found",
		})
		return
	}

	if lock.RepoID != repo.ID() {
		renderJSON(w, r, http.StatusNotFound, lfs.ErrorResponse{
			Message: "lock not found",
		})
		return
	}

	owner, err := datastore.GetUserByID(ctx, dbx, lock.UserID)
	if err != nil {
		logger.Error("error getting lock owner", "err", err)
		renderJSON(w, r, http.StatusInternalServerError, lfs.ErrorResponse{
			Message: "internal server error",
		})
		return
	}

	l := lfs.Lock{
		ID:       strconv.FormatInt(lock.ID, 10),
		Path:     lock.Path,
		LockedAt: lock.CreatedAt,
		Owner: lfs.Owner{
			Name: owner.Username,
		},
	}

	// Retrieve user context first for authorization checks
	user := proto.UserFromContext(ctx)
	if user == nil {
		logger.Error("error getting user from context")
		renderJSON(w, r, http.StatusUnauthorized, lfs.ErrorResponse{
			Message: "unauthorized",
		})
		return
	}

	// Force delete another user's lock (requires admin privileges)
	if req.Force {
		if !user.IsAdmin() {
			logger.Error("non-admin user attempted force delete", "user", user.Username())
			renderJSON(w, r, http.StatusForbidden, lfs.ErrorResponse{
				Message: "admin access required for force delete",
			})
			return
		}

		if err := datastore.DeleteLFSLock(ctx, dbx, repo.ID(), lockID); err != nil {
			logger.Error("error deleting lock", "err", err)
			renderJSON(w, r, http.StatusInternalServerError, lfs.ErrorResponse{
				Message: "internal server error",
			})
			return
		}

		renderJSON(w, r, http.StatusOK, l)
		return
	}

	// Delete our own lock - verify ownership
	if owner.ID != user.ID() {
		logger.Error("error deleting another user's lock")
		renderJSON(w, r, http.StatusForbidden, lfs.ErrorResponse{
			Message: "lock belongs to another user",
		})
		return
	}

	if err := datastore.DeleteLFSLock(ctx, dbx, repo.ID(), lockID); err != nil {
		logger.Error("error deleting lock", "err", err)
		renderJSON(w, r, http.StatusInternalServerError, lfs.ErrorResponse{
			Message: "internal server error",
		})
		return
	}

	renderJSON(w, r, http.StatusOK, lfs.LockResponse{Lock: l})
}

// renderJSON renders a JSON response with the given status code and value. It
// also sets the Content-Type header to the JSON LFS media type (application/vnd.git-lfs+json).
func renderJSON(w http.ResponseWriter, r *http.Request, statusCode int, v interface{}) {
	hdrLfs(w)
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.FromContext(r.Context()).Error("error encoding json", "err", err)
	}
}

func renderNotAcceptable(w http.ResponseWriter) {
	renderStatus(http.StatusNotAcceptable)(w, nil)
}

func isLfs(r *http.Request) bool {
	contentType := r.Header.Get("Content-Type")
	accept := r.Header.Get("Accept")
	return strings.HasPrefix(contentType, lfs.MediaType) && strings.HasPrefix(accept, lfs.MediaType)
}

func isBinary(r *http.Request) bool {
	contentType := r.Header.Get("Content-Type")
	return strings.HasPrefix(contentType, "application/octet-stream")
}

func hdrLfs(w http.ResponseWriter) {
	w.Header().Set("Content-Type", lfs.MediaType)
	w.Header().Set("Accept", lfs.MediaType)
}
