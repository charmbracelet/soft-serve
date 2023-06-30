package db

import (
	"context"
	"errors"
	"sync"
)

var (
	registry = map[string]Database{}
	mtx      = sync.Mutex{}

	// ErrNotFound is returned when a database is not found.
	ErrNotFound = errors.New("database not found")
)

// Register registers a database.
func Register(name string, db Database) {
	mtx.Lock()
	defer mtx.Unlock()
	registry[name] = db
}

// Get returns a database.
func Get(name string) Database {
	mtx.Lock()
	defer mtx.Unlock()
	return registry[name]
}

// List returns a list of registered databases.
func List() []string {
	mtx.Lock()
	defer mtx.Unlock()
	dbs := make([]string, 0)
	for name := range registry {
		dbs = append(dbs, name)
	}
	return dbs
}

// New returns a new database.
func New(ctx context.Context, name string, dataSource string) (Database, error) {
	mtx.Lock()
	defer mtx.Unlock()
	db := registry[name]
	if db == nil {
		return nil, ErrNotFound
	}
	return db.Open(ctx, dataSource)
}
