package config

import (
	"os"
	"testing"

	"github.com/matryer/is"
)

func TestParseMultipleKeys(t *testing.T) {
	is := is.New(t)
	td := t.TempDir()
	is.NoErr(os.Setenv("SOFT_SERVE_INITIAL_ADMIN_KEYS", "testdata/k1.pub\nssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFxIobhwtfdwN7m1TFt9wx3PsfvcAkISGPxmbmbauST8 a@b"))
	is.NoErr(os.Setenv("SOFT_SERVE_DATA_PATH", td))
	t.Cleanup(func() {
		is.NoErr(os.Unsetenv("SOFT_SERVE_INITIAL_ADMIN_KEYS"))
		is.NoErr(os.Unsetenv("SOFT_SERVE_DATA_PATH"))
	})
	cfg := DefaultConfig()
	is.NoErr(cfg.ParseEnv())
	is.Equal(cfg.InitialAdminKeys, []string{
		"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAINMwLvyV3ouVrTysUYGoJdl5Vgn5BACKov+n9PlzfPwH",
		"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFxIobhwtfdwN7m1TFt9wx3PsfvcAkISGPxmbmbauST8",
	})
}

func TestMergeInitAdminKeys(t *testing.T) {
	is := is.New(t)
	is.NoErr(os.Setenv("SOFT_SERVE_INITIAL_ADMIN_KEYS", "testdata/k1.pub"))
	t.Cleanup(func() { is.NoErr(os.Unsetenv("SOFT_SERVE_INITIAL_ADMIN_KEYS")) })
	cfg := &Config{
		DataPath:         t.TempDir(),
		InitialAdminKeys: []string{"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFxIobhwtfdwN7m1TFt9wx3PsfvcAkISGPxmbmbauST8 a@b"},
	}
	is.NoErr(cfg.WriteConfig())
	is.NoErr(cfg.Parse())
	is.Equal(cfg.InitialAdminKeys, []string{
		"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAINMwLvyV3ouVrTysUYGoJdl5Vgn5BACKov+n9PlzfPwH",
		"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFxIobhwtfdwN7m1TFt9wx3PsfvcAkISGPxmbmbauST8",
	})
}

func TestValidateInitAdminKeys(t *testing.T) {
	is := is.New(t)
	cfg := &Config{
		DataPath: t.TempDir(),
		InitialAdminKeys: []string{
			"testdata/k1.pub",
			"abc",
			"",
		},
	}
	is.NoErr(cfg.WriteConfig())
	is.NoErr(cfg.Parse())
	is.Equal(cfg.InitialAdminKeys, []string{
		"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAINMwLvyV3ouVrTysUYGoJdl5Vgn5BACKov+n9PlzfPwH",
	})
}

func TestCustomConfigLocation(t *testing.T) {
	is := is.New(t)
	td := t.TempDir()
	t.Cleanup(func() {
		is.NoErr(os.Unsetenv("SOFT_SERVE_CONFIG_LOCATION"))
	})

	// Test that we get data from the custom file location, and not from the data dir.
	is.NoErr(os.Setenv("SOFT_SERVE_CONFIG_LOCATION", "testdata/config.yaml"))
	is.NoErr(os.Setenv("SOFT_SERVE_DATA_PATH", td))
	cfg := DefaultConfig()
	is.NoErr(cfg.Parse())
	is.Equal(cfg.Name, "Test server name")
	// If we unset the custom location, then use the default location.
	is.NoErr(os.Unsetenv("SOFT_SERVE_CONFIG_LOCATION"))
	cfg = DefaultConfig()
	is.Equal(cfg.Name, "Soft Serve")
	// Test that if the custom config location doesn't exist, default to datapath config.
	is.NoErr(os.Setenv("SOFT_SERVE_CONFIG_LOCATION", "testdata/config_nonexistent.yaml"))
	cfg = DefaultConfig()
	is.Equal(cfg.Name, "Soft Serve")
}

func TestParseMultipleHeaders(t *testing.T) {
	is := is.New(t)
	is.NoErr(os.Setenv("SOFT_SERVE_HTTP_CORS_ALLOWED_HEADERS", "Accept,Accept-Language,User-Agent"))
	t.Cleanup(func() {
		is.NoErr(os.Unsetenv("SOFT_SERVE_HTTP_CORS_ALLOWED_HEADERS"))
	})
	cfg := DefaultConfig()
	is.NoErr(cfg.ParseEnv())
	is.Equal(cfg.HTTP.CORS.AllowedHeaders, []string{
		"Accept",
		"Accept-Language",
		"User-Agent",
	})
}

func TestParseMultipleOrigins(t *testing.T) {
	is := is.New(t)
	is.NoErr(os.Setenv("SOFT_SERVE_HTTP_CORS_ALLOWED_ORIGINS", "http://example.com,https://example.com"))
	t.Cleanup(func() {
		is.NoErr(os.Unsetenv("SOFT_SERVE_HTTP_CORS_ALLOWED_ORIGINS"))
	})
	cfg := DefaultConfig()
	is.NoErr(cfg.ParseEnv())
	is.Equal(cfg.HTTP.CORS.AllowedOrigins, []string{
		"http://localhost:23232",
		"http://example.com",
		"https://example.com",
	})
}

func TestParseMultipleMethods(t *testing.T) {
	is := is.New(t)
	is.NoErr(os.Setenv("SOFT_SERVE_HTTP_CORS_ALLOWED_METHODS", "GET,POST,PUT"))
	t.Cleanup(func() {
		is.NoErr(os.Unsetenv("SOFT_SERVE_HTTP_CORS_ALLOWED_METHODS"))
	})
	cfg := DefaultConfig()
	is.NoErr(cfg.ParseEnv())
	is.Equal(cfg.HTTP.CORS.AllowedMethods, []string{
		"GET",
		"POST",
		"PUT",
	})
}
