package database

import (
	"context"

	"github.com/charmbracelet/log/v2"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/store"
)

type datastore struct {
	ctx    context.Context
	cfg    *config.Config
	db     *db.DB
	logger *log.Logger

	*settingsStore
	*repoStore
	*userStore
	*collabStore
	*lfsStore
	*accessTokenStore
	*webhookStore
}

// New returns a new store.Store database.
func New(ctx context.Context, db *db.DB) store.Store {
	cfg := config.FromContext(ctx)
	logger := log.FromContext(ctx).WithPrefix("store")

	s := &datastore{
		ctx:    ctx,
		cfg:    cfg,
		db:     db,
		logger: logger,

		settingsStore:    &settingsStore{},
		repoStore:        &repoStore{},
		userStore:        &userStore{},
		collabStore:      &collabStore{},
		lfsStore:         &lfsStore{},
		accessTokenStore: &accessTokenStore{},
	}

	return s
}
