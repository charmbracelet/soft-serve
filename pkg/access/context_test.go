package access

import (
	"context"
	"testing"
)

func TestGoodFromContext(t *testing.T) {
	ctx := WithContext(context.TODO(), AdminAccess)
	if ac := FromContext(ctx); ac != AdminAccess {
		t.Errorf("FromContext(ctx) => %d, want %d", ac, AdminAccess)
	}
}

func TestBadFromContext(t *testing.T) {
	ctx := context.TODO()
	if ac := FromContext(ctx); ac != -1 {
		t.Errorf("FromContext(ctx) => %d, want %d", ac, -1)
	}
}
