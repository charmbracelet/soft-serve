package db

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"
)

func TestWrapErrorBadNoRows(t *testing.T) {
	for _, e := range []error{
		fmt.Errorf("foo"),
		errors.New("bar"),
	} {
		if err := WrapError(e); err != e {
			t.Errorf("WrapError(%v) => %v, want %v", e, err, e)
		}
	}
}

func TestWrapErrorGoodNoRows(t *testing.T) {
	if err := WrapError(sql.ErrNoRows); err != ErrRecordNotFound {
		t.Errorf("WrapError(sql.ErrNoRows) => %v, want %v", err, ErrRecordNotFound)
	}
}
