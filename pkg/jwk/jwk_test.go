package jwk

import (
	"errors"
	"testing"

	"github.com/charmbracelet/soft-serve/pkg/config"
)

func TestBadNewPair(t *testing.T) {
	_, err := NewPair(nil)
	if !errors.Is(err, config.ErrNilConfig) {
		t.Errorf("NewPair(nil) => %v, want %v", err, config.ErrNilConfig)
	}
}

func TestGoodNewPair(t *testing.T) {
	cfg := config.DefaultConfig()
	if _, err := NewPair(cfg); err != nil {
		t.Errorf("NewPair(cfg) => _, %v, want nil error", err)
	}
}
