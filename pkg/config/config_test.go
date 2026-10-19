package config

import (
	"os"
	"testing"

	"github.com/charmbracelet/soft-serve/pkg/access"
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

func TestDefaultRepoDisabledByDefault(t *testing.T) {
	is := is.New(t)
	is.Equal(DefaultConfig().DefaultRepo, "")
}

func TestParseDefaultRepo(t *testing.T) {
	is := is.New(t)
	is.NoErr(os.Setenv("SOFT_SERVE_DEFAULT_REPO", "gitops"))
	t.Cleanup(func() {
		is.NoErr(os.Unsetenv("SOFT_SERVE_DEFAULT_REPO"))
	})
	cfg := DefaultConfig()
	is.NoErr(cfg.ParseEnv())
	is.Equal(cfg.DefaultRepo, "gitops")
}

func TestParseDefaultRepoFromFile(t *testing.T) {
	is := is.New(t)
	cfg := &Config{DataPath: t.TempDir()}
	is.NoErr(parseFile(cfg, "testdata/config_default_repo.yaml"))
	is.Equal(cfg.DefaultRepo, "gitops")
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

func TestAnonAccessEnvUnsetByDefault(t *testing.T) {
	is := is.New(t)
	cfg := DefaultConfig()
	is.NoErr(cfg.ParseEnv())
	// nil is the "no override" sentinel.
	is.True(cfg.AnonAccess == nil)
}

func TestParseAnonAccessEnv(t *testing.T) {
	is := is.New(t)
	is.NoErr(os.Setenv("SOFT_SERVE_ANON_ACCESS", access.AdminAccess.String()))
	t.Cleanup(func() {
		is.NoErr(os.Unsetenv("SOFT_SERVE_ANON_ACCESS"))
	})
	cfg := DefaultConfig()
	is.NoErr(cfg.ParseEnv())
	is.True(cfg.AnonAccess != nil)
	is.Equal(*cfg.AnonAccess, access.AdminAccess)
}

// An invalid anon-access level is rejected at parse time by AccessLevel's
// TextUnmarshaler, so it never reaches Validate as a bad value.
func TestParseRejectsInvalidAnonAccess(t *testing.T) {
	is := is.New(t)
	is.NoErr(os.Setenv("SOFT_SERVE_ANON_ACCESS", "not-a-real-access-level"))
	t.Cleanup(func() {
		is.NoErr(os.Unsetenv("SOFT_SERVE_ANON_ACCESS"))
	})
	cfg := DefaultConfig()
	err := cfg.ParseEnv()
	is.True(err != nil)
}

func TestAllowKeylessEnvUnsetByDefault(t *testing.T) {
	is := is.New(t)
	cfg := DefaultConfig()
	is.NoErr(cfg.ParseEnv())
	// nil is the "no override" sentinel — distinct from an explicit false.
	is.True(cfg.AllowKeyless == nil)
}

func TestParseAllowKeylessEnvTrue(t *testing.T) {
	is := is.New(t)
	is.NoErr(os.Setenv("SOFT_SERVE_ALLOW_KEYLESS", "true"))
	t.Cleanup(func() {
		is.NoErr(os.Unsetenv("SOFT_SERVE_ALLOW_KEYLESS"))
	})
	cfg := DefaultConfig()
	is.NoErr(cfg.ParseEnv())
	is.True(cfg.AllowKeyless != nil)
	is.Equal(*cfg.AllowKeyless, true)
}

func TestParseAllowKeylessEnvFalse(t *testing.T) {
	is := is.New(t)
	is.NoErr(os.Setenv("SOFT_SERVE_ALLOW_KEYLESS", "false"))
	t.Cleanup(func() {
		is.NoErr(os.Unsetenv("SOFT_SERVE_ALLOW_KEYLESS"))
	})
	cfg := DefaultConfig()
	is.NoErr(cfg.ParseEnv())
	// Explicit false must survive as a real override, not collapse back to
	// "unset" — that's the whole reason this field is a *bool.
	is.True(cfg.AllowKeyless != nil)
	is.Equal(*cfg.AllowKeyless, false)
}

func TestValidateClientStrictHostKeyChecking(t *testing.T) {
	for _, tt := range []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"", StrictHostKeyCheckingAcceptNew, false},
		{"accept-new", StrictHostKeyCheckingAcceptNew, false},
		{"yes", StrictHostKeyCheckingYes, false},
		// "no" is exactly the setting being removed; "ask" cannot work
		// without a terminal.
		{"no", "", true},
		{"ask", "", true},
		{"off", "", true},
	} {
		cfg := DefaultConfig()
		cfg.DataPath = t.TempDir()
		cfg.SSH.ClientStrictHostKeyChecking = tt.in
		err := cfg.Validate()
		if tt.wantErr {
			if err == nil {
				t.Errorf("Validate() with %q: expected error", tt.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("Validate() with %q: %v", tt.in, err)
		}
		if cfg.SSH.ClientStrictHostKeyChecking != tt.want {
			t.Errorf("Validate() with %q = %q, want %q", tt.in, cfg.SSH.ClientStrictHostKeyChecking, tt.want)
		}
	}
}

func TestValidateRejectsUnquotableDataPath(t *testing.T) {
	for _, dp := range []string{`/data/"quoted"`, "/data/line\nbreak", "/data/cr\r"} {
		cfg := DefaultConfig()
		cfg.DataPath = dp
		if err := cfg.Validate(); err == nil {
			t.Errorf("Validate() accepted data path %q", dp)
		}
	}

	cfg := DefaultConfig()
	cfg.DataPath = t.TempDir() + "/Application Support/soft serve"
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() rejected a data path with spaces: %v", err)
	}
}

func TestValidateParsesAllowedHosts(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DataPath = t.TempDir()
	if len(cfg.SSH.AllowedHosts()) != 0 {
		t.Fatal("allowlist populated before Validate")
	}

	cfg.SSH.ClientAllowedHosts = []string{"git.internal.example", "", "10.0.0.0/8:2222", " "}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	if got := cfg.SSH.ClientAllowedHosts; len(got) != 2 {
		t.Errorf("blank entries kept: %q", got)
	}
	if got := cfg.SSH.AllowedHosts(); len(got) != 2 {
		t.Errorf("parsed %d entries, want 2", len(got))
	}
	if err := cfg.SSH.AllowedHosts().Match("git.internal.example", ""); err != nil {
		t.Errorf("parsed allowlist does not match its own entry: %v", err)
	}

	cfg.SSH.ClientAllowedHosts = []string{"10.0.0.0/99"}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() accepted an invalid network")
	}
}
