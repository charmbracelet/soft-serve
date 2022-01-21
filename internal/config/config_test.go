package config

import (
	"testing"

	"github.com/charmbracelet/soft-serve/config"
	"github.com/matryer/is"
)

func TestConfigInitialKeys(t *testing.T) {
	cfg, err := NewConfig(&config.Config{
		RepoPath: t.TempDir(),
		KeyPath:  t.TempDir(),
		InitialAdminKeys: []string{
			"testdata/k1.pub",
			"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFxIobhwtfdwN7m1TFt9wx3PsfvcAkISGPxmbmbauST8 a@b",
		},
	})
	is := is.New(t)
	is.NoErr(err)
	is.Equal(cfg.Users[0].PublicKeys, []string{
		"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAINMwLvyV3ouVrTysUYGoJdl5Vgn5BACKov+n9PlzfPwH a@b",
		"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFxIobhwtfdwN7m1TFt9wx3PsfvcAkISGPxmbmbauST8 a@b",
	})
}
