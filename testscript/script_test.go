package testscript

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/soft-serve/server"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/test"
	"github.com/rogpeppe/go-internal/testscript"
)

func TestScript(t *testing.T) {
	key, err := filepath.Abs("./testdata/admin1")
	if err != nil {
		t.Fatal(err)
	}

	testscript.Run(t, testscript.Params{
		Dir:           "testdata/script",
		UpdateScripts: true,
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"soft": func(ts *testscript.TestScript, _ bool, args []string) {
				args = append([]string{
					"-F", "/dev/null",
					"-o", "StrictHostKeyChecking=no",
					"-o", "UserKnownHostsFile=/dev/null",
					"-o", "IdentityAgent=none",
					"-o", "IdentitiesOnly=yes",
					"-i", key,
					"-p", ts.Getenv("SSH_PORT"),
					"localhost",
					"--",
				}, args...)
				ts.Check(ts.Exec("ssh", args...))
			},
		},
		Setup: func(e *testscript.Env) error {
			cfg := config.DefaultConfig()
			cfg.InitialAdminKeys = []string{
				"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIJI/1tawpdPmzuJcTGTJ+QReqB6cRUdKj4iQIdJUFdrl",
			}
			cfg.DataPath = t.TempDir()

			sshPort := test.RandomPort()
			e.Setenv("SSH_PORT", fmt.Sprintf("%d", sshPort))
			cfg.SSH.ListenAddr = fmt.Sprintf("localhost:%d", sshPort)
			cfg.HTTP.ListenAddr = fmt.Sprintf("localhost:%d", test.RandomPort())
			cfg.Git.ListenAddr = fmt.Sprintf("localhost:%d", test.RandomPort())
			cfg.Stats.ListenAddr = fmt.Sprintf("localhost:%d", test.RandomPort())
			ctx := config.WithContext(context.Background(), cfg)
			srv, err := server.NewServer(ctx)
			if err != nil {
				return err
			}
			go func() {
				if err := srv.Start(); err != nil {
					e.T().Fatal(err)
				}
			}()
			e.Defer(func() {
				if err := srv.Shutdown(context.Background()); err != nil {
					e.T().Fatal(err)
				}
			})
			return nil
		},
	})
}
