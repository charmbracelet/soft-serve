package db

import (
	"context"
	"database/sql"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/jmoiron/sqlx"
)

func trace(l *log.Logger, query string, args ...interface{}) {
	if l != nil {
		// Remove newlines and tabs
		query = strings.ReplaceAll(query, "\t", "")
		query = strings.TrimSpace(query)
		l.Debug("trace", "query", query, "args", args)
	}
}

// Select is a wrapper around sqlx.Select that logs the query and arguments.
func (d *DB) Select(dest interface{}, query string, args ...interface{}) error {
	trace(d.logger, query, args...)
	return d.DB.Select(dest, query, args...)
}

// Get is a wrapper around sqlx.Get that logs the query and arguments.
func (d *DB) Get(dest interface{}, query string, args ...interface{}) error {
	trace(d.logger, query, args...)
	return d.DB.Get(dest, query, args...)
}

// Queryx is a wrapper around sqlx.Queryx that logs the query and arguments.
func (d *DB) Queryx(query string, args ...interface{}) (*sqlx.Rows, error) {
	trace(d.logger, query, args...)
	return d.DB.Queryx(query, args...)
}

// QueryRowx is a wrapper around sqlx.QueryRowx that logs the query and arguments.
func (d *DB) QueryRowx(query string, args ...interface{}) *sqlx.Row {
	trace(d.logger, query, args...)
	return d.DB.QueryRowx(query, args...)
}

// Exec is a wrapper around sqlx.Exec that logs the query and arguments.
func (d *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	trace(d.logger, query, args...)
	return d.DB.Exec(query, args...)
}

// SelectContext is a wrapper around sqlx.SelectContext that logs the query and arguments.
func (d *DB) SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	trace(d.logger, query, args...)
	return d.DB.SelectContext(ctx, dest, query, args...)
}

// GetContext is a wrapper around sqlx.GetContext that logs the query and arguments.
func (d *DB) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	trace(d.logger, query, args...)
	return d.DB.GetContext(ctx, dest, query, args...)
}

// QueryxContext is a wrapper around sqlx.QueryxContext that logs the query and arguments.
func (d *DB) QueryxContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error) {
	trace(d.logger, query, args...)
	return d.DB.QueryxContext(ctx, query, args...)
}

// QueryRowxContext is a wrapper around sqlx.QueryRowxContext that logs the query and arguments.
func (d *DB) QueryRowxContext(ctx context.Context, query string, args ...interface{}) *sqlx.Row {
	trace(d.logger, query, args...)
	return d.DB.QueryRowxContext(ctx, query, args...)
}

// ExecContext is a wrapper around sqlx.ExecContext that logs the query and arguments.
func (d *DB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	trace(d.logger, query, args...)
	return d.DB.ExecContext(ctx, query, args...)
}

// Select is a wrapper around sqlx.Select that logs the query and arguments.
func (t *Tx) Select(dest interface{}, query string, args ...interface{}) error {
	trace(t.logger, query, args...)
	return t.Tx.Select(dest, query, args...)
}

// Get is a wrapper around sqlx.Get that logs the query and arguments.
func (t *Tx) Get(dest interface{}, query string, args ...interface{}) error {
	trace(t.logger, query, args...)
	return t.Tx.Get(dest, query, args...)
}

// Queryx is a wrapper around sqlx.Queryx that logs the query and arguments.
func (t *Tx) Queryx(query string, args ...interface{}) (*sqlx.Rows, error) {
	trace(t.logger, query, args...)
	return t.Tx.Queryx(query, args...)
}

// QueryRowx is a wrapper around sqlx.QueryRowx that logs the query and arguments.
func (t *Tx) QueryRowx(query string, args ...interface{}) *sqlx.Row {
	trace(t.logger, query, args...)
	return t.Tx.QueryRowx(query, args...)
}

// Exec is a wrapper around sqlx.Exec that logs the query and arguments.
func (t *Tx) Exec(query string, args ...interface{}) (sql.Result, error) {
	trace(t.logger, query, args...)
	return t.Tx.Exec(query, args...)
}

// SelectContext is a wrapper around sqlx.SelectContext that logs the query and arguments.
func (t *Tx) SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	trace(t.logger, query, args...)
	return t.Tx.SelectContext(ctx, dest, query, args...)
}

// GetContext is a wrapper around sqlx.GetContext that logs the query and arguments.
func (t *Tx) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	trace(t.logger, query, args...)
	return t.Tx.GetContext(ctx, dest, query, args...)
}

// QueryxContext is a wrapper around sqlx.QueryxContext that logs the query and arguments.
func (t *Tx) QueryxContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error) {
	trace(t.logger, query, args...)
	return t.Tx.QueryxContext(ctx, query, args...)
}

// QueryRowxContext is a wrapper around sqlx.QueryRowxContext that logs the query and arguments.
func (t *Tx) QueryRowxContext(ctx context.Context, query string, args ...interface{}) *sqlx.Row {
	trace(t.logger, query, args...)
	return t.Tx.QueryRowxContext(ctx, query, args...)
}

// ExecContext is a wrapper around sqlx.ExecContext that logs the query and arguments.
func (t *Tx) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	trace(t.logger, query, args...)
	return t.Tx.ExecContext(ctx, query, args...)
}
