package config

import "testing"

func TestNewConfigFile(t *testing.T) {
	for _, cfg := range []*Config{
		nil,
		DefaultConfig(),
		{},
	} {
		if s := newConfigFile(cfg); s == "" {
			t.Errorf("newConfigFile(nil) => %q, want non-empty string", s)
		}
	}
}
