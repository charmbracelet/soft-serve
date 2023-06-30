package sqlite

import (
	"errors"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/server/db"
	"github.com/jmoiron/sqlx"
	"golang.org/x/net/context"
)

var (
	// Options database options for the sqlite database.
	Options = map[string][]string{
		"_pragma": {
			"busy_timeout(5000)",
			"foreign_keys(1)",
		},
	}

	// ErrDuplicateKey is returned when a unique constraint is violated.
	ErrDuplicateKey = errors.New("record already exists")

	// ErrNoRecord is returned when a record is not found.
	ErrNoRecord = errors.New("record not found")
)

func init() {
	db.Register("sqlite", &Sqlite{})
}

// Sqlite is the interface that wraps basic sqlite operations.
type Sqlite struct {
	ctx    context.Context
	db     *sqlx.DB
	logger *log.Logger
}

var _ db.Database = (*Sqlite)(nil)

// DBx returns the underlying sqlx database.
func (s *Sqlite) DBx() *sqlx.DB {
	return s.db
}

// Open opens a new sqlite database connection.
func (s *Sqlite) Open(ctx context.Context, path string) (db.Database, error) {
	logger := log.FromContext(ctx).WithPrefix("sqlite")
	dataSource := path
	if len(Options) > 0 {
		dataSource += "?"
		for k, v := range Options {
			for i, o := range v {
				dataSource += k + "=" + o
				if i < len(v)-1 {
					dataSource += "&"
				}
			}
		}
	}
	db, err := sqlx.ConnectContext(ctx, "sqlite", dataSource)
	if err != nil {
		return nil, err
	}

	return &Sqlite{
		ctx:    ctx,
		db:     db,
		logger: logger,
	}, nil
}

// Close closes the sqlite database connection.
func (s *Sqlite) Close() error {
	return s.db.Close()
}
