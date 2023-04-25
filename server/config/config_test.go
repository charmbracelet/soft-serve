package config

import (
	"os"
	"testing"

	"github.com/matryer/is"
)

func TestParseMultipleKeys(t *testing.T) {
	is := is.New(t)
	is.NoErr(os.Setenv("SOFT_SERVE_INITIAL_ADMIN_KEYS", "testdata/k1.pub\ntestdata/k2.pub"))
	t.Cleanup(func() { is.NoErr(os.Unsetenv("SOFT_SERVE_INITIAL_ADMIN_KEYS")) })
	cfg := DefaultConfig()
	is.Equal(cfg.InitialAdminKeys, []string{
		"testdata/k1.pub",
		"testdata/k2.pub",
	})
}
