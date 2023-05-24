package testscript

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
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
	var lock sync.Mutex

	t.Setenv("SOFT_SERVE_TEST_NO_HOOKS", "1")

	// we'll use this key to talk with soft serve, and since testscript changes
	// the cwd, we need to get its full path here
	key, err := filepath.Abs("./testdata/admin1")
	if err != nil {
		t.Fatal(err)
	}

	// git does not handle 0600, and on clone, will save the files with its
	// default perm, 0644, which is too open for ssh.
	for _, f := range []string{
		"admin1",
		"admin2",
		"user1",
		"user2",
	} {
		if err := os.Chmod(filepath.Join("./testdata/", f), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	sshArgs := []string{
		"-F", "/dev/null",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "IdentityAgent=none",
		"-o", "IdentitiesOnly=yes",
		"-o", "ServerAliveInterval=60",
		"-i", key,
	}

	check := func(ts *testscript.TestScript, err error, neg bool) {
		if neg && err == nil {
			ts.Fatalf("expected error, got nil")
		}
		if !neg {
			ts.Check(err)
		}
	}

	testscript.Run(t, testscript.Params{
		Dir:           "testdata/script",
		UpdateScripts: *update,
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"soft": func(ts *testscript.TestScript, neg bool, args []string) {
				args = append(
					sshArgs,
					append([]string{
						"-p", ts.Getenv("SSH_PORT"),
						"localhost",
						"--",
					}, args...)...,
				)
				if runtime.GOOS == "windows" {
					cmd := exec.Command("ssh", args...)
					out, err := cmd.CombinedOutput()
					ts.Logf("RUNNING %v: output: %s error: %v", cmd.Args, string(out), err)
				}
				check(ts, ts.Exec("ssh", args...), neg)
			},
			"git": func(ts *testscript.TestScript, neg bool, args []string) {
				ts.Setenv(
					"GIT_SSH_COMMAND",
					strings.Join(append([]string{"ssh"}, sshArgs...), " "),
				)
				args = append([]string{
					"-c", "user.email=john@example.com",
					"-c", "user.name=John Doe",
				}, args...)
				check(ts, ts.Exec("git", args...), neg)
			},
			"mkreadme": func(ts *testscript.TestScript, neg bool, args []string) {
				if len(args) != 1 {
					ts.Fatalf("must have exactly 1 arg, the filename, got %d", len(args))
				}
				check(ts, os.WriteFile(ts.MkAbs(args[0]), []byte("# example\ntest project"), 0o644), neg)
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

			// prevent race condition in lipgloss...
			// this will probably be autofixed when we start using the colors
			// from the ssh session instead of the server.
			// XXX: take another look at this soon
			lock.Lock()
			srv, err := server.NewServer(ctx)
			if err != nil {
				return err
			}
			lock.Unlock()

			go func() {
				if err := srv.Start(); err != nil {
					e.T().Fatal(err)
				}
			}()

			e.Defer(func() {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				if err := srv.Shutdown(ctx); err != nil {
					e.T().Fatal(err)
				}
			})

			// wait until the server is up
			for {
				conn, _ := net.DialTimeout(
					"tcp",
					net.JoinHostPort("localhost", fmt.Sprintf("%d", sshPort)),
					time.Second,
				)
				if conn != nil {
					conn.Close()
					break
				}
			}

			return nil
		},
	})
}
