package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"  // postgres driver
	_ "modernc.org/sqlite" // sqlite driver
)

// DB is the interface for a Soft Serve database.
type DB struct {
	*sqlx.DB
	logger *log.Logger
}

// Open opens a database connection.
func Open(ctx context.Context, driverName string, dsn string) (*DB, error) {
	db, err := sqlx.ConnectContext(ctx, driverName, dsn)
	if err != nil {
		return nil, err
	}

	d := &DB{
		DB: db,
	}

	if config.IsVerbose() {
		logger := log.FromContext(ctx).WithPrefix("db")
		d.logger = logger
	}

	return d, nil
}

// Close implements db.DB.
func (d *DB) Close() error {
	return d.DB.Close()
}

// Tx is a database transaction.
type Tx struct {
	*sqlx.Tx
	logger *log.Logger
}

// Transaction implements db.DB.
func (d *DB) Transaction(fn func(tx *Tx) error) error {
	return d.TransactionContext(context.Background(), fn)
}

// TransactionContext implements db.DB.
func (d *DB) TransactionContext(ctx context.Context, fn func(tx *Tx) error) error {
	txx, err := d.DB.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	tx := &Tx{txx, d.logger}
	if err := fn(tx); err != nil {
		return rollback(tx, err)
	}

	if err := tx.Commit(); err != nil {
		if errors.Is(err, sql.ErrTxDone) {
			// this is ok because whoever did finish the tx should have also written the error already.
			return nil
		}
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func rollback(tx *Tx, err error) error {
	if rerr := tx.Rollback(); rerr != nil {
		if errors.Is(rerr, sql.ErrTxDone) {
			return err
		}
		return fmt.Errorf("failed to rollback: %s: %w", err.Error(), rerr)
	}

	return err
}
