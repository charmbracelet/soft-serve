package testscript

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/keygen"
	"github.com/charmbracelet/soft-serve/server"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/test"
	"github.com/rogpeppe/go-internal/testscript"
	"golang.org/x/crypto/ssh"
)

var update = flag.Bool("update", false, "update script files")

func TestScript(t *testing.T) {
	flag.Parse()
	var lock sync.Mutex

	mkkey := func(name string) (string, *keygen.SSHKeyPair) {
		path := filepath.Join(t.TempDir(), name)
		pair, err := keygen.New(path, keygen.WithKeyType(keygen.Ed25519), keygen.WithWrite())
		if err != nil {
			t.Fatal(err)
		}
		return path, pair
	}

	key, admin1 := mkkey("admin1")
	_, admin2 := mkkey("admin2")
	_, user1 := mkkey("user1")

	testscript.Run(t, testscript.Params{
		Dir:           "./testdata/",
		UpdateScripts: *update,
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"soft":     cmdSoft(admin1.Signer()),
			"git":      cmdGit(key),
			"mkreadme": cmdMkReadme,
			"dos2unix": cmdDos2Unix,
		},
		Setup: func(e *testscript.Env) error {
			sshPort := test.RandomPort()
			e.Setenv("SSH_PORT", fmt.Sprintf("%d", sshPort))
			e.Setenv("ADMIN1_AUTHORIZED_KEY", admin1.AuthorizedKey())
			e.Setenv("ADMIN2_AUTHORIZED_KEY", admin2.AuthorizedKey())
			e.Setenv("USER1_AUTHORIZED_KEY", user1.AuthorizedKey())
			data := t.TempDir()
			cfg := config.Config{
				Name:             "Test Soft Serve",
				DataPath:         data,
				InitialAdminKeys: []string{admin1.AuthorizedKey()},
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
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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

func cmdSoft(key ssh.Signer) func(ts *testscript.TestScript, neg bool, args []string) {
	return func(ts *testscript.TestScript, neg bool, args []string) {
		cli, err := ssh.Dial(
			"tcp",
			net.JoinHostPort("localhost", ts.Getenv("SSH_PORT")),
			&ssh.ClientConfig{
				User:            "admin",
				Auth:            []ssh.AuthMethod{ssh.PublicKeys(key)},
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			},
		)
		ts.Check(err)
		defer cli.Close()

		sess, err := cli.NewSession()
		ts.Check(err)
		defer sess.Close()

		sess.Stdout = ts.Stdout()
		sess.Stderr = ts.Stderr()

		check(ts, sess.Run(strings.Join(args, " ")), neg)
	}
}

// P.S. Windows sucks!
func cmdDos2Unix(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("unsupported: ! dos2unix")
	}
	if len(args) < 1 {
		ts.Fatalf("usage: dos2unix paths...")
	}
	for _, arg := range args {
		filename := ts.MkAbs(arg)
		data, err := os.ReadFile(filename)
		if err != nil {
			ts.Fatalf("%s: %v", filename, err)
		}

		// Replace all '\r\n' with '\n'.
		data = bytes.ReplaceAll(data, []byte{'\r', '\n'}, []byte{'\n'})

		if err := os.WriteFile(filename, data, 0o644); err != nil {
			ts.Fatalf("%s: %v", filename, err)
		}
	}
}

func cmdGit(key string) func(ts *testscript.TestScript, neg bool, args []string) {
	return func(ts *testscript.TestScript, neg bool, args []string) {
		sshArgs := []string{
			"-o", "StrictHostKeyChecking=no",
			"-o", "IdentityAgent=none",
			"-o", "IdentitiesOnly=yes",
			"-o", "ServerAliveInterval=60",
			// Escape the key path for Windows.
			"-i", strings.ReplaceAll(key, `\`, `\\`),
		}
		// Windows null device
		// https://stackoverflow.com/a/36746090/10913628
		if runtime.GOOS == "windows" {
			sshArgs = append(sshArgs, []string{
				"-F", `$nul`,
				"-o", `UserKnownHostsFile=$nul`,
			}...)
		} else {
			sshArgs = append(sshArgs, []string{
				"-F", "/dev/null",
				"-o", "UserKnownHostsFile=/dev/null",
			}...)
		}
		ts.Setenv(
			"GIT_SSH_COMMAND",
			strings.Join(append([]string{"ssh"}, sshArgs...), " "),
		)
		args = append([]string{
			"-c", "user.email=john@example.com",
			"-c", "user.name=John Doe",
		}, args...)
		check(ts, ts.Exec("git", args...), neg)
	}
}

func cmdMkReadme(ts *testscript.TestScript, neg bool, args []string) {
	if len(args) != 1 {
		ts.Fatalf("usage: mkreadme path")
	}
	content := []byte("# example\ntest project")
	check(ts, os.WriteFile(ts.MkAbs(args[0]), content, 0o644), neg)
}

func check(ts *testscript.TestScript, err error, neg bool) {
	if neg && err == nil {
		ts.Fatalf("expected error, got nil")
	}
	if !neg {
		ts.Check(err)
	}
}
