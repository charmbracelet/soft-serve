package daemon

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/migrate"
	"github.com/charmbracelet/soft-serve/pkg/git"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/store"
	"github.com/charmbracelet/soft-serve/pkg/store/database"
	"github.com/charmbracelet/soft-serve/pkg/test"
	"github.com/go-git/go-git/v5/plumbing/format/pktline"
	_ "modernc.org/sqlite" // sqlite driver
)

var (
	testDaemon  *GitDaemon
	testBackend *backend.Backend
)

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "soft-serve-test")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	ctx := context.TODO()
	cfg := config.DefaultConfig()
	cfg.DataPath = tmp
	cfg.Git.MaxConnections = 3
	cfg.Git.MaxTimeout = 100
	cfg.Git.IdleTimeout = 1
	cfg.Git.ListenAddr = fmt.Sprintf(":%d", test.RandomPort())
	if err := cfg.Validate(); err != nil {
		log.Fatal(err)
	}
	ctx = config.WithContext(ctx, cfg)
	dbx, err := db.Open(ctx, cfg.DB.Driver, cfg.DB.DataSource)
	if err != nil {
		log.Fatal(err)
	}
	defer dbx.Close() //nolint: errcheck
	if err := migrate.Migrate(ctx, dbx); err != nil {
		log.Fatal(err)
	}
	datastore := database.New(ctx, dbx)
	ctx = store.WithContext(ctx, datastore)
	be := backend.New(ctx, cfg, dbx, datastore)
	testBackend = be
	ctx = backend.WithContext(ctx, be)
	d, err := NewGitDaemon(ctx)
	if err != nil {
		log.Fatal(err)
	}
	testDaemon = d
	go d.ListenAndServe() //nolint:errcheck
	code := m.Run()
	os.Unsetenv("SOFT_SERVE_DATA_PATH")
	os.Unsetenv("SOFT_SERVE_GIT_MAX_CONNECTIONS")
	os.Unsetenv("SOFT_SERVE_GIT_MAX_TIMEOUT")
	os.Unsetenv("SOFT_SERVE_GIT_IDLE_TIMEOUT")
	os.Unsetenv("SOFT_SERVE_GIT_LISTEN_ADDR")
	_ = d.Close()
	_ = dbx.Close()
	os.Exit(code)
}

func TestIdleTimeout(t *testing.T) {
	var err error
	var c net.Conn
	var tries int
	var dialer net.Dialer
	for {
		c, err = dialer.DialContext(t.Context(), "tcp", testDaemon.addr)
		if err != nil && tries >= 3 {
			t.Fatalf("failed to connect to daemon after %d tries: %v", tries, err)
		}
		tries++
		if testDaemon.conns.Size() != 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(2 * time.Second)
	_, err = readPktline(c)
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}

// requestUploadPack asks the daemon for a repository and returns the first
// line it answers with.
func requestUploadPack(t *testing.T, repo string) string {
	t.Helper()
	c, err := net.Dial("tcp", testDaemon.addr) //nolint:noctx
	if err != nil {
		t.Fatalf("failed to connect to daemon: %v", err)
	}
	defer c.Close() //nolint:errcheck
	if err := pktline.NewEncoder(c).EncodeString("git-upload-pack /" + repo + ".git\x00"); err != nil {
		t.Fatalf("expected nil, got error: %v", err)
	}
	// The scanner reports an ERR packet as an error carrying its message.
	line, err := readPktline(c)
	if err != nil {
		return err.Error()
	}
	return line
}

// TestInvalidRepo verifies the daemon, which is always anonymous, answers a
// missing repository and a private one identically so neither is revealed.
func TestInvalidRepo(t *testing.T) {
	ctx := context.Background()
	if _, err := testBackend.CreateRepository(ctx, "private", nil, proto.RepositoryOptions{Private: true}); err != nil {
		t.Fatalf("failed to create private repo: %v", err)
	}

	if _, err := testBackend.CreateRepository(ctx, "public", nil, proto.RepositoryOptions{}); err != nil {
		t.Fatalf("failed to create public repo: %v", err)
	}

	want := git.ErrNotAuthed.Error()
	missing := requestUploadPack(t, "test")
	private := requestUploadPack(t, "private")
	if missing != want {
		t.Errorf("missing repo: expected %q, got %q", want, missing)
	}
	if private != want {
		t.Errorf("private repo: expected %q, got %q", want, private)
	}

	// A path that escapes the repositories directory gets the same answer.
	if got := requestUploadPack(t, "../escape"); got != want {
		t.Errorf("escaping path: expected %q, got %q", want, got)
	}

	// A public repository is served: the first line is not a refusal.
	public := requestUploadPack(t, "public")
	if public == want || public == git.ErrInvalidRepo.Error() {
		t.Errorf("public repo was refused: %q", public)
	}
}

func readPktline(c net.Conn) (string, error) {
	pktout := pktline.NewScanner(c)
	if !pktout.Scan() {
		return "", pktout.Err()
	}
	return strings.TrimSpace(string(pktout.Bytes())), nil
}
