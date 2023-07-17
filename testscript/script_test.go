package testscript

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/keygen"
	"github.com/charmbracelet/soft-serve/server"
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/db"
	"github.com/charmbracelet/soft-serve/server/db/migrate"
	"github.com/charmbracelet/soft-serve/server/test"
	"github.com/rogpeppe/go-internal/testscript"
	"golang.org/x/crypto/ssh"
	_ "modernc.org/sqlite" // sqlite Driver
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
			"usoft":    cmdSoft(user1.Signer()),
			"git":      cmdGit(key),
			"mkfile":   cmdMkfile,
			"dos2unix": cmdDos2Unix,
		},
		Setup: func(e *testscript.Env) error {
			data := t.TempDir()

			sshPort := test.RandomPort()
			sshListen := fmt.Sprintf("localhost:%d", sshPort)
			gitPort := test.RandomPort()
			gitListen := fmt.Sprintf("localhost:%d", gitPort)
			httpPort := test.RandomPort()
			httpListen := fmt.Sprintf("localhost:%d", httpPort)
			statsPort := test.RandomPort()
			statsListen := fmt.Sprintf("localhost:%d", statsPort)
			serverName := "Test Soft Serve"

			e.Setenv("SSH_PORT", fmt.Sprintf("%d", sshPort))
			e.Setenv("ADMIN1_AUTHORIZED_KEY", admin1.AuthorizedKey())
			e.Setenv("ADMIN2_AUTHORIZED_KEY", admin2.AuthorizedKey())
			e.Setenv("USER1_AUTHORIZED_KEY", user1.AuthorizedKey())
			e.Setenv("SSH_KNOWN_HOSTS_FILE", filepath.Join(t.TempDir(), "known_hosts"))
			e.Setenv("SSH_KNOWN_CONFIG_FILE", filepath.Join(t.TempDir(), "config"))

			cfg := config.DefaultConfig()
			cfg.DataPath = data
			cfg.Name = serverName
			cfg.InitialAdminKeys = []string{admin1.AuthorizedKey()}
			cfg.SSH.ListenAddr = sshListen
			cfg.SSH.PublicURL = "ssh://" + sshListen
			cfg.Git.ListenAddr = gitListen
			cfg.HTTP.ListenAddr = httpListen
			cfg.HTTP.PublicURL = "http://" + httpListen
			cfg.Stats.ListenAddr = statsListen
			cfg.DB.Driver = "sqlite"

			if err := cfg.Validate(); err != nil {
				return err
			}

			ctx := config.WithContext(context.Background(), cfg)

			// TODO: test postgres
			dbx, err := db.Open(ctx, cfg.DB.Driver, cfg.DB.DataSource)
			if err != nil {
				return fmt.Errorf("open database: %w", err)
			}

			if err := migrate.Migrate(ctx, dbx); err != nil {
				return fmt.Errorf("migrate database: %w", err)
			}

			ctx = db.WithContext(ctx, dbx)
			be := backend.New(ctx, cfg, dbx)
			ctx = backend.WithContext(ctx, be)

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
				defer dbx.Close() // nolint: errcheck
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

var sshConfig = `
Host *
  UserKnownHostsFile %q
  StrictHostKeyChecking no
  IdentityAgent none
  IdentitiesOnly yes
  ServerAliveInterval 60
`

func cmdGit(key string) func(ts *testscript.TestScript, neg bool, args []string) {
	return func(ts *testscript.TestScript, neg bool, args []string) {
		ts.Check(os.WriteFile(
			ts.Getenv("SSH_KNOWN_CONFIG_FILE"),
			[]byte(fmt.Sprintf(sshConfig, ts.Getenv("SSH_KNOWN_HOSTS_FILE"))),
			0o600,
		))
		sshArgs := []string{
			"-F", filepath.ToSlash(ts.Getenv("SSH_KNOWN_CONFIG_FILE")),
			"-i", filepath.ToSlash(key),
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

func cmdMkfile(ts *testscript.TestScript, neg bool, args []string) {
	if len(args) < 2 {
		ts.Fatalf("usage: mkfile path content")
	}
	check(ts, os.WriteFile(
		ts.MkAbs(args[0]),
		[]byte(strings.Join(args[1:], " ")),
		0o644,
	), neg)
}

func check(ts *testscript.TestScript, err error, neg bool) {
	if neg && err == nil {
		ts.Fatalf("expected error, got nil")
	}
	if !neg {
		ts.Check(err)
	}
}
