package db_test

import (
	"context"
	"testing"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/internal/test"
)

func TestBadFromContext(t *testing.T) {
	ctx := context.TODO()
	if c := db.FromContext(ctx); c != nil {
		t.Errorf("FromContext(ctx) => %v, want %v", c, nil)
	}
}

func TestGoodFromContext(t *testing.T) {
	ctx := context.TODO()
	dbx, err := test.OpenSqlite(ctx, t)
	if err != nil {
		t.Fatal(err)
	}
	ctx = db.WithContext(ctx, dbx)
	if c := db.FromContext(ctx); c == nil {
		t.Errorf("FromContext(ctx) => %v, want %v", c, dbx)
	}
}
