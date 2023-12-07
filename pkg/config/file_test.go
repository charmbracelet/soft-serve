package config

import "testing"

func TestNewConfigFile(t *testing.T) {
	for _, cfg := range []*Config{
		nil,
		DefaultConfig(),
		&Config{},
	} {
		if s := newConfigFile(cfg); s == "" {
			t.Errorf("newConfigFile(nil) => %q, want non-empty string", s)
		}
	}
}
