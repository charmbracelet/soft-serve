package testscript

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/keygen"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/test"
	"github.com/rogpeppe/go-internal/testscript"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

var (
	update  = flag.Bool("update", false, "update script files")
	binPath string
)

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "soft-serve*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temporary directory: %s", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmp)

	binPath = filepath.Join(tmp, "soft")
	if runtime.GOOS == "windows" {
		binPath += ".exe"
	}

	// Build the soft binary with -cover flag.
	cmd := exec.Command("go", "build", "-coverpkg=./...", "-cover", "-o", binPath, filepath.Join("..", "cmd", "soft"))
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build soft-serve binary: %s", err)
		os.Exit(1)
	}

	// Run tests
	os.Exit(m.Run())

	// Add binPath to PATH
	os.Setenv("PATH", fmt.Sprintf("%s%c%s", os.Getenv("PATH"), os.PathListSeparator, filepath.Dir(binPath)))
}

func TestScript(t *testing.T) {
	flag.Parse()

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
		Dir:                 "./testdata/",
		UpdateScripts:       *update,
		RequireExplicitExec: true,
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"soft":          cmdSoft(admin1.Signer()),
			"usoft":         cmdSoft(user1.Signer()),
			"git":           cmdGit(key),
			"curl":          cmdCurl,
			"mkfile":        cmdMkfile,
			"envfile":       cmdEnvfile,
			"readfile":      cmdReadfile,
			"dos2unix":      cmdDos2Unix,
			"new-webhook":   cmdNewWebhook,
			"waitforserver": cmdWaitforserver,
		},
		Setup: func(e *testscript.Env) error {
			// Add binPath to PATH
			e.Setenv("PATH", fmt.Sprintf("%s%c%s", filepath.Dir(binPath), os.PathListSeparator, e.Getenv("PATH")))

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

			e.Setenv("DATA_PATH", data)
			e.Setenv("SSH_PORT", fmt.Sprintf("%d", sshPort))
			e.Setenv("HTTP_PORT", fmt.Sprintf("%d", httpPort))
			e.Setenv("ADMIN1_AUTHORIZED_KEY", admin1.AuthorizedKey())
			e.Setenv("ADMIN2_AUTHORIZED_KEY", admin2.AuthorizedKey())
			e.Setenv("USER1_AUTHORIZED_KEY", user1.AuthorizedKey())
			e.Setenv("SSH_KNOWN_HOSTS_FILE", filepath.Join(t.TempDir(), "known_hosts"))
			e.Setenv("SSH_KNOWN_CONFIG_FILE", filepath.Join(t.TempDir(), "config"))

			// Soft Serve debug environment variables
			for _, env := range []string{
				"SOFT_SERVE_DEBUG",
				"SOFT_SERVE_VERBOSE",
			} {
				if v, ok := os.LookupEnv(env); ok {
					e.Setenv(env, v)
				}
			}

			// TODO: test different configs
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
			cfg.LFS.Enabled = true
			// cfg.LFS.SSHEnabled = true

			// Parse os SOFT_SERVE environment variables
			if err := cfg.ParseEnv(); err != nil {
				return err
			}

			// Override the database data source if we're using postgres
			// so we can create a temporary database for the tests.
			if cfg.DB.Driver == "postgres" {
				err, cleanup := setupPostgres(e.T(), cfg)
				if err != nil {
					return err
				}
				if cleanup != nil {
					e.Defer(cleanup)
				}
			}

			for _, env := range cfg.Environ() {
				parts := strings.SplitN(env, "=", 2)
				if len(parts) != 2 {
					e.T().Fatal("invalid environment variable", env)
				}
				e.Setenv(parts[0], parts[1])
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
		// Disable git prompting for credentials.
		ts.Setenv("GIT_TERMINAL_PROMPT", "0")
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

func cmdReadfile(ts *testscript.TestScript, neg bool, args []string) {
	ts.Stdout().Write([]byte(ts.ReadFile(args[0])))
}

func cmdEnvfile(ts *testscript.TestScript, neg bool, args []string) {
	if len(args) < 1 {
		ts.Fatalf("usage: envfile key=file...")
	}

	for _, arg := range args {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			ts.Fatalf("usage: envfile key=file...")
		}
		key := parts[0]
		file := parts[1]
		ts.Setenv(key, strings.TrimSpace(ts.ReadFile(file)))
	}
}

func cmdNewWebhook(ts *testscript.TestScript, neg bool, args []string) {
	type webhookSite struct {
		UUID string `json:"uuid"`
	}

	if len(args) != 1 {
		ts.Fatalf("usage: new-webhook <env-name>")
	}

	const whSite = "https://webhook.site"
	req, err := http.NewRequest(http.MethodPost, whSite+"/token", nil)
	check(ts, err, neg)

	resp, err := http.DefaultClient.Do(req)
	check(ts, err, neg)

	defer resp.Body.Close()
	var site webhookSite
	check(ts, json.NewDecoder(resp.Body).Decode(&site), neg)

	ts.Setenv(args[0], whSite+"/"+site.UUID)
}

func cmdCurl(ts *testscript.TestScript, neg bool, args []string) {
	var verbose bool
	var headers []string
	var data string
	method := http.MethodGet

	cmd := &cobra.Command{
		Use:  "curl",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url, err := url.Parse(args[0])
			if err != nil {
				return err
			}

			req, err := http.NewRequest(method, url.String(), nil)
			if err != nil {
				return err
			}

			if data != "" {
				req.Body = io.NopCloser(strings.NewReader(data))
			}

			if verbose {
				fmt.Fprintf(cmd.ErrOrStderr(), "< %s %s\n", req.Method, url.String())
			}

			for _, header := range headers {
				parts := strings.SplitN(header, ":", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid header: %s", header)
				}
				req.Header.Add(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
			}

			if userInfo := url.User; userInfo != nil {
				password, _ := userInfo.Password()
				req.SetBasicAuth(userInfo.Username(), password)
			}

			if verbose {
				for key, values := range req.Header {
					for _, value := range values {
						fmt.Fprintf(cmd.ErrOrStderr(), "< %s: %s\n", key, value)
					}
				}
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}

			if verbose {
				fmt.Fprintf(ts.Stderr(), "> %s\n", resp.Status)
				for key, values := range resp.Header {
					for _, value := range values {
						fmt.Fprintf(cmd.ErrOrStderr(), "> %s: %s\n", key, value)
					}
				}
			}

			defer resp.Body.Close()
			buf, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			cmd.Print(string(buf))

			return nil
		},
	}

	cmd.SetArgs(args)
	cmd.SetOut(ts.Stdout())
	cmd.SetErr(ts.Stderr())

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", verbose, "verbose")
	cmd.Flags().StringArrayVarP(&headers, "header", "H", nil, "HTTP header")
	cmd.Flags().StringVarP(&method, "request", "X", method, "HTTP method")
	cmd.Flags().StringVarP(&data, "data", "d", data, "HTTP data")

	check(ts, cmd.Execute(), neg)
}

func cmdWaitforserver(ts *testscript.TestScript, neg bool, args []string) {
	// wait until the server is up
	for {
		conn, _ := net.DialTimeout(
			"tcp",
			net.JoinHostPort("localhost", fmt.Sprintf("%s", ts.Getenv("SSH_PORT"))),
			time.Second,
		)
		if conn != nil {
			conn.Close()
			break
		}
	}
}

func setupPostgres(t testscript.T, cfg *config.Config) (error, func()) {
	// Indicates postgres
	// Create a disposable database
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	dbName := fmt.Sprintf("softserve_test_%d", rnd.Int63())
	dbDsn := cfg.DB.DataSource
	if dbDsn == "" {
		cfg.DB.DataSource = "postgres://postgres@localhost:5432/postgres?sslmode=disable"
	}

	dbUrl, err := url.Parse(cfg.DB.DataSource)
	if err != nil {
		return err, nil
	}

	scheme := dbUrl.Scheme
	if scheme == "" {
		scheme = "postgres"
	}

	host := dbUrl.Hostname()
	if host == "" {
		host = "localhost"
	}

	connInfo := fmt.Sprintf("host=%s sslmode=disable", host)
	username := dbUrl.User.Username()
	if username != "" {
		connInfo += fmt.Sprintf(" user=%s", username)
		password, ok := dbUrl.User.Password()
		if ok {
			username = fmt.Sprintf("%s:%s", username, password)
			connInfo += fmt.Sprintf(" password=%s", password)
		}
		username = fmt.Sprintf("%s@", username)
	} else {
		connInfo += " user=postgres"
		username = "postgres@"
	}

	port := dbUrl.Port()
	if port != "" {
		connInfo += fmt.Sprintf(" port=%s", port)
		port = fmt.Sprintf(":%s", port)
	}

	cfg.DB.DataSource = fmt.Sprintf("%s://%s%s%s/%s?sslmode=disable",
		scheme,
		username,
		host,
		port,
		dbName,
	)

	// Create the database
	dbx, err := db.Open(context.TODO(), cfg.DB.Driver, connInfo)
	if err != nil {
		return err, nil
	}

	if _, err := dbx.Exec("CREATE DATABASE " + dbName); err != nil {
		return err, nil
	}

	return nil, func() {
		dbx, err := db.Open(context.TODO(), cfg.DB.Driver, connInfo)
		if err != nil {
			t.Fatal("failed to open database", dbName, err)
		}

		if _, err := dbx.Exec("DROP DATABASE " + dbName); err != nil {
			t.Fatal("failed to drop database", dbName, err)
		}
	}
}
