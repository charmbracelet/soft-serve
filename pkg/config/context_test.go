package config

import (
	"context"
	"reflect"
	"testing"
)

func TestBadFromContext(t *testing.T) {
	ctx := context.TODO()
	if c := FromContext(ctx); c != nil {
		t.Errorf("FromContext(ctx) => %v, want %v", c, nil)
	}
}

func TestGoodFromContext(t *testing.T) {
	ctx := WithContext(context.TODO(), &Config{})
	if c := FromContext(ctx); c == nil {
		t.Errorf("FromContext(ctx) => %v, want %v", c, &Config{})
	}
}

func TestGoodFromContextWithDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	ctx := WithContext(context.TODO(), cfg)
	if c := FromContext(ctx); c == nil || !reflect.DeepEqual(c, cfg) {
		t.Errorf("FromContext(ctx) => %v, want %v", c, cfg)
	}
}
