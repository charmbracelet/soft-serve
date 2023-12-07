package db

import (
	"context"
	"strings"
	"testing"
)

func TestOpenUnknownDriver(t *testing.T) {
	_, err := Open(context.TODO(), "invalid", "")
	if err == nil {
		t.Error("Open(invalid) => nil, want error")
	}
	if !strings.Contains(err.Error(), "unknown driver") {
		t.Errorf("Open(invalid) => %v, want error containing 'unknown driver'", err)
	}
}
