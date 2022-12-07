package daemon

import (
	"bytes"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"testing"

	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/git"
	"github.com/go-git/go-git/v5/plumbing/format/pktline"
)

var testDaemon *Daemon

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "soft-serve-test")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	os.Setenv("SOFT_SERVE_DATA_PATH", tmp)
	os.Setenv("SOFT_SERVE_ANON_ACCESS", "read-only")
	os.Setenv("SOFT_SERVE_GIT_MAX_CONNECTIONS", "3")
	os.Setenv("SOFT_SERVE_GIT_MAX_TIMEOUT", "100")
	os.Setenv("SOFT_SERVE_GIT_IDLE_TIMEOUT", "3")
	os.Setenv("SOFT_SERVE_GIT_PORT", strconv.Itoa(randomPort()))
	cfg := config.DefaultConfig()
	d, err := NewDaemon(cfg)
	if err != nil {
		log.Fatal(err)
	}
	testDaemon = d
	go func() {
		if err := d.Start(); err != ErrServerClosed {
			log.Fatal(err)
		}
	}()
	defer d.Close()
	os.Exit(m.Run())
	os.Unsetenv("SOFT_SERVE_DATA_PATH")
	os.Unsetenv("SOFT_SERVE_ANON_ACCESS")
	os.Unsetenv("SOFT_SERVE_GIT_MAX_CONNECTIONS")
	os.Unsetenv("SOFT_SERVE_GIT_MAX_TIMEOUT")
	os.Unsetenv("SOFT_SERVE_GIT_IDLE_TIMEOUT")
	os.Unsetenv("SOFT_SERVE_GIT_PORT")
}

func TestIdleTimeout(t *testing.T) {
	c, err := net.Dial("tcp", testDaemon.addr)
	if err != nil {
		t.Fatal(err)
	}
	out, err := readPktline(c)
	if err != nil {
		t.Fatalf("expected nil, got error: %v", err)
	}
	if out != git.ErrTimeout.Error() {
		t.Fatalf("expected %q error, got %q", git.ErrTimeout, out)
	}
}

func TestInvalidRepo(t *testing.T) {
	c, err := net.Dial("tcp", testDaemon.addr)
	if err != nil {
		t.Fatal(err)
	}
	if err := pktline.NewEncoder(c).EncodeString("git-upload-pack /test.git\x00"); err != nil {
		t.Fatalf("expected nil, got error: %v", err)
	}
	out, err := readPktline(c)
	if err != nil {
		t.Fatalf("expected nil, got error: %v", err)
	}
	if out != git.ErrInvalidRepo.Error() {
		t.Fatalf("expected %q error, got %q", git.ErrInvalidRepo, out)
	}
}

func readPktline(c net.Conn) (string, error) {
	buf, err := io.ReadAll(c)
	if err != nil {
		return "", err
	}
	pktout := pktline.NewScanner(bytes.NewReader(buf))
	if !pktout.Scan() {
		return "", pktout.Err()
	}
	return string(pktout.Bytes()), nil
}

func randomPort() int {
	addr, _ := net.Listen("tcp", ":0") //nolint:gosec
	_ = addr.Close()
	return addr.Addr().(*net.TCPAddr).Port
}
