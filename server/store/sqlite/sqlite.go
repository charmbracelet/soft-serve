package filesqlite

import (
	"context"
	"errors"
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/server/cache"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/db"
	"github.com/charmbracelet/soft-serve/server/db/sqlite"
	"github.com/charmbracelet/soft-serve/server/store"
	"github.com/go-git/go-billy/v5"
)

var (
	// ErrDbNotSqlite is returned when the database is not a SQLite database.
	ErrDbNotSqlite = errors.New("database is not a SQLite database")

	// ErrRepoNotExist is returned when a repository does not exist.
	ErrRepoNotExist = fmt.Errorf("repository does not exist")

	// ErrRepoExist is returned when a repository already exists.
	ErrRepoExist = fmt.Errorf("repository already exists")
)

// SqliteStore is a file-based SQLite store.
type SqliteStore struct {
	fs     billy.Filesystem
	ctx    context.Context
	cfg    *config.Config
	db     db.Database
	cache  cache.Cache
	logger *log.Logger
}

var _ store.Store = (*SqliteStore)(nil)

func init() {
	store.Register("sqlite", newFileSqliteStore)
}

func newFileSqliteStore(ctx context.Context, fs billy.Filesystem) (store.Store, error) {
	sdb := db.FromContext(ctx)
	if sdb == nil {
		return nil, db.ErrNoDatabase
	}

	if _, ok := sdb.(*sqlite.Sqlite); !ok {
		return nil, ErrDbNotSqlite
	}

	ss := &SqliteStore{
		fs:     fs,
		ctx:    ctx,
		db:     sdb,
		logger: log.FromContext(ctx).WithPrefix("filesqlite"),
		cfg:    config.FromContext(ctx),
		cache:  cache.FromContext(ctx),
	}

	c := cache.FromContext(ctx)
	if c == nil {
		return nil, cache.ErrNotFound
	}

	return ss, nil
}

func (ss *SqliteStore) Filesystem() billy.Filesystem {
	return ss.fs
}

func (ss *SqliteStore) reposPath() string {
	return ss.fs.Root()
}

// cacheKey returns the cache key for a repository.
func cacheKey(name string) string {
	return fmt.Sprintf("repo:%s", name)
}
