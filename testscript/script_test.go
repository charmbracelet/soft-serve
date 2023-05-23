package testscript

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/soft-serve/server"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/test"
	"github.com/rogpeppe/go-internal/testscript"
)

var update = flag.Bool("update", false, "update script files")

func TestScript(t *testing.T) {
	flag.Parse()
	key, err := filepath.Abs("./testdata/admin1")
	if err != nil {
		t.Fatal(err)
	}

	testscript.Run(t, testscript.Params{
		Dir:           "testdata/script",
		UpdateScripts: *update,
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
			sshPort := test.RandomPort()
			e.Setenv("SSH_PORT", fmt.Sprintf("%d", sshPort))
			data := t.TempDir()
			cfg := config.Config{
				Name:     "Test Soft Serve",
				DataPath: data,
				InitialAdminKeys: []string{
					"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIJI/1tawpdPmzuJcTGTJ+QReqB6cRUdKj4iQIdJUFdrl",
				},
				SSH: config.SSHConfig{
					ListenAddr:    fmt.Sprintf("localhost:%d", sshPort),
					PublicURL:     fmt.Sprintf("ssh://localhost:%d", sshPort),
					KeyPath:       filepath.Join(data, "ssh", "soft_serve_host_ed25519"),
					ClientKeyPath: filepath.Join(data, "ssh", "soft_serve_client_ed25519"),
				},
				Git: config.GitConfig{
					ListenAddr:     fmt.Sprintf("localhost:%d", test.RandomPort()),
					IdleTimeout:    3,
					MaxConnections: 32,
				},
				HTTP: config.HTTPConfig{
					ListenAddr: fmt.Sprintf("localhost:%d", test.RandomPort()),
					PublicURL:  fmt.Sprintf("http://localhost:%d", test.RandomPort()),
				},
				Stats: config.StatsConfig{
					ListenAddr: fmt.Sprintf("localhost:%d", test.RandomPort()),
				},
				Log: config.LogConfig{
					Format:     "text",
					TimeFormat: time.DateTime,
				},
			}
			ctx := config.WithContext(context.Background(), &cfg)
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
