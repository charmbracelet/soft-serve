package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matryer/is"
	"gopkg.in/yaml.v3"
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
	is.Equal(cfg.InitialAdminKeys, []string{
		"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAINMwLvyV3ouVrTysUYGoJdl5Vgn5BACKov+n9PlzfPwH",
		"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFxIobhwtfdwN7m1TFt9wx3PsfvcAkISGPxmbmbauST8",
	})
}

func TestMergeInitAdminKeys(t *testing.T) {
	is := is.New(t)
	is.NoErr(os.Setenv("SOFT_SERVE_INITIAL_ADMIN_KEYS", "testdata/k1.pub"))
	t.Cleanup(func() { is.NoErr(os.Unsetenv("SOFT_SERVE_INITIAL_ADMIN_KEYS")) })
	bts, err := yaml.Marshal(&Config{
		InitialAdminKeys: []string{"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFxIobhwtfdwN7m1TFt9wx3PsfvcAkISGPxmbmbauST8 a@b"},
	})
	is.NoErr(err)
	fp := filepath.Join(t.TempDir(), "config.yaml")
	err = os.WriteFile(fp, bts, 0644)
	is.NoErr(err)
	cfg, err := ParseConfig(fp)
	is.NoErr(err)
	is.Equal(cfg.InitialAdminKeys, []string{
		"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAINMwLvyV3ouVrTysUYGoJdl5Vgn5BACKov+n9PlzfPwH",
		"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFxIobhwtfdwN7m1TFt9wx3PsfvcAkISGPxmbmbauST8",
	})
}

func TestValidateInitAdminKeys(t *testing.T) {
	is := is.New(t)
	bts, err := yaml.Marshal(&Config{
		InitialAdminKeys: []string{
			"testdata/k1.pub",
			"abc",
			"",
		},
	})
	is.NoErr(err)
	fp := filepath.Join(t.TempDir(), "config.yaml")
	err = os.WriteFile(fp, bts, 0644)
	is.NoErr(err)
	cfg, err := ParseConfig(fp)
	is.NoErr(err)
	is.Equal(cfg.InitialAdminKeys, []string{
		"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAINMwLvyV3ouVrTysUYGoJdl5Vgn5BACKov+n9PlzfPwH",
	})
}
