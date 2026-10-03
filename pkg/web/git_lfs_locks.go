package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"charm.land/log/v2"
	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/lfs"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/store"
	"github.com/gorilla/mux"
)

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

	ctx := r.Context()
	logger := log.FromContext(ctx).WithPrefix("http.lfs-locks")

	// Checked in withAccess too. Repeated here so a routing or middleware
	// mistake cannot turn into an unauthorized lock.
	if access.FromContext(ctx) < access.ReadWriteAccess {
		askCredentials(w, r)
		renderJSON(w, http.StatusForbidden, lfs.ErrorResponse{
			Message: "write access required",
		})
		return
	}

	var req lfs.LockCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("error decoding json", "err", err)
		renderJSON(w, http.StatusBadRequest, lfs.ErrorResponse{
			Message: "invalid request: " + err.Error(),
		})
		return
	}

	repo, ok := lfsRepository(w, r)
	if !ok {
		return
	}

	user := proto.UserFromContext(ctx)
	if user == nil {
		logger.Error("error getting user from context")
		renderJSON(w, http.StatusNotFound, lfs.ErrorResponse{
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
						renderJSON(w, http.StatusInternalServerError, lfs.ErrorResponse{
							Message: "internal server error",
						})
						return
					}
					lockOwner.Name = owner.Username
				}
				errResp.Lock.Owner = lockOwner
			}
			renderJSON(w, http.StatusConflict, errResp)
			return
		}
		logger.Error("error creating lock", "err", err)
		renderJSON(w, http.StatusInternalServerError, lfs.ErrorResponse{
			Message: "internal server error",
		})
		return
	}

	lock, err := datastore.GetLFSLockForUserPath(ctx, dbx, repo.ID(), user.ID(), req.Path)
	if err != nil {
		logger.Error("error getting lock", "err", err)
		renderJSON(w, http.StatusInternalServerError, lfs.ErrorResponse{
			Message: "internal server error",
		})
		return
	}

	renderJSON(w, http.StatusCreated, lfs.LockResponse{
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

	logger := log.FromContext(ctx).WithPrefix("http.lfs-locks")
	dbx := db.FromContext(ctx)
	datastore := store.FromContext(ctx)
	repo, ok := lfsRepository(w, r)
	if !ok {
		return
	}

	if id > 0 {
		lock, err := datastore.GetLFSLockByID(ctx, dbx, repo.ID(), id)
		if err != nil {
			if errors.Is(err, db.ErrRecordNotFound) {
				renderJSON(w, http.StatusNotFound, lfs.ErrorResponse{
					Message: "lock not found",
				})
				return
			}
			logger.Error("error getting lock", "err", err)
			renderJSON(w, http.StatusInternalServerError, lfs.ErrorResponse{
				Message: "internal server error",
			})
			return
		}

		owner, err := datastore.GetUserByID(ctx, dbx, lock.UserID)
		if err != nil {
			logger.Error("error getting lock owner", "err", err)
			renderJSON(w, http.StatusInternalServerError, lfs.ErrorResponse{
				Message: "internal server error",
			})
			return
		}

		renderJSON(w, http.StatusOK, lfs.LockListResponse{
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
				renderJSON(w, http.StatusNotFound, lfs.ErrorResponse{
					Message: "lock not found",
				})
				return
			}
			logger.Error("error getting lock", "err", err)
			renderJSON(w, http.StatusInternalServerError, lfs.ErrorResponse{
				Message: "internal server error",
			})
			return
		}

		owner, err := datastore.GetUserByID(ctx, dbx, lock.UserID)
		if err != nil {
			logger.Error("error getting lock owner", "err", err)
			renderJSON(w, http.StatusInternalServerError, lfs.ErrorResponse{
				Message: "internal server error",
			})
			return
		}

		renderJSON(w, http.StatusOK, lfs.LockListResponse{
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
		renderJSON(w, http.StatusInternalServerError, lfs.ErrorResponse{
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
				renderJSON(w, http.StatusInternalServerError, lfs.ErrorResponse{
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

	renderJSON(w, http.StatusOK, resp)
}

// POST: /<repo>.git/info/lfs/objects/locks/verify
func serviceLfsLocksVerify(w http.ResponseWriter, r *http.Request) {
	if !isLfs(r) {
		renderNotAcceptable(w)
		return
	}

	ctx := r.Context()
	logger := log.FromContext(ctx).WithPrefix("http.lfs-locks")
	repo, ok := lfsRepository(w, r)
	if !ok {
		return
	}

	var req lfs.LockVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("error decoding request", "err", err)
		renderJSON(w, http.StatusBadRequest, lfs.ErrorResponse{
			Message: "invalid request: " + err.Error(),
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
		renderJSON(w, http.StatusInternalServerError, lfs.ErrorResponse{
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
				renderJSON(w, http.StatusInternalServerError, lfs.ErrorResponse{
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

	renderJSON(w, http.StatusOK, resp)
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
		renderJSON(w, http.StatusBadRequest, lfs.ErrorResponse{
			Message: "invalid request",
		})
		return
	}

	lockID, err := strconv.ParseInt(lockIDStr, 10, 64)
	if err != nil {
		logger.Error("error parsing lock id", "err", err)
		renderJSON(w, http.StatusBadRequest, lfs.ErrorResponse{
			Message: "invalid request",
		})
		return
	}

	var req lfs.LockDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("error decoding request", "err", err)
		renderJSON(w, http.StatusBadRequest, lfs.ErrorResponse{
			Message: "invalid request: " + err.Error(),
		})
		return
	}

	dbx := db.FromContext(ctx)
	datastore := store.FromContext(ctx)
	repo, ok := lfsRepository(w, r)
	if !ok {
		return
	}

	// The lock being deleted. Looked up scoped to this repository so that a
	// lock ID belonging to another repository cannot be read through it.
	lock, err := datastore.GetLFSLockByID(ctx, dbx, repo.ID(), lockID)
	if err != nil {
		logger.Error("error getting lock", "err", err)
		renderJSON(w, http.StatusNotFound, lfs.ErrorResponse{
			Message: "lock not found",
		})
		return
	}

	owner, err := datastore.GetUserByID(ctx, dbx, lock.UserID)
	if err != nil {
		logger.Error("error getting lock owner", "err", err)
		renderJSON(w, http.StatusInternalServerError, lfs.ErrorResponse{
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
		renderJSON(w, http.StatusUnauthorized, lfs.ErrorResponse{
			Message: "unauthorized",
		})
		return
	}

	// Force delete another user's lock (requires admin privileges)
	if req.Force {
		if !user.IsAdmin() {
			logger.Error("non-admin user attempted force delete", "user", user.Username())
			renderJSON(w, http.StatusForbidden, lfs.ErrorResponse{
				Message: "admin access required for force delete",
			})
			return
		}

		if err := datastore.DeleteLFSLock(ctx, dbx, repo.ID(), lockID); err != nil {
			logger.Error("error deleting lock", "err", err)
			renderJSON(w, http.StatusInternalServerError, lfs.ErrorResponse{
				Message: "internal server error",
			})
			return
		}

		renderJSON(w, http.StatusOK, l)
		return
	}

	// Delete our own lock - verify ownership
	if owner.ID != user.ID() {
		logger.Error("error deleting another user's lock")
		renderJSON(w, http.StatusForbidden, lfs.ErrorResponse{
			Message: "lock belongs to another user",
		})
		return
	}

	if err := datastore.DeleteLFSLock(ctx, dbx, repo.ID(), lockID); err != nil {
		logger.Error("error deleting lock", "err", err)
		renderJSON(w, http.StatusInternalServerError, lfs.ErrorResponse{
			Message: "internal server error",
		})
		return
	}

	renderJSON(w, http.StatusOK, lfs.LockResponse{Lock: l})
}
