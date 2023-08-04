package backend

import (
	"context"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/db"
	"github.com/charmbracelet/soft-serve/server/store"
	"github.com/charmbracelet/soft-serve/server/task"
)

// Backend is the Soft Serve backend that handles users, repositories, and
// server settings management and operations.
type Backend struct {
	ctx     context.Context
	cfg     *config.Config
	db      *db.DB
	store   store.Store
	logger  *log.Logger
	cache   *cache
	manager *task.Manager
}

// New returns a new Soft Serve backend.
func New(ctx context.Context, cfg *config.Config, db *db.DB) *Backend {
	dbstore := store.FromContext(ctx)
	logger := log.FromContext(ctx).WithPrefix("backend")
	b := &Backend{
		ctx:     ctx,
		cfg:     cfg,
		db:      db,
		store:   dbstore,
		logger:  logger,
		manager: task.NewManager(ctx),
	}

	// TODO: implement a proper caching interface
	cache := newCache(b, 1000)
	b.cache = cache

	return b
}
