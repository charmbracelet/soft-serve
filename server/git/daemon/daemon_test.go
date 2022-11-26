package daemon

import (
	"bytes"
	"io"
	"log"
	"net"
	"os"
	"testing"

	appCfg "github.com/charmbracelet/soft-serve/config"
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
	cfg := &config.Config{
		Host:     "",
		DataPath: tmp,
		Git: config.GitConfig{
			// Reduce the max timeout to 100 second so we can test the timeout.
			MaxTimeout: 100,
			// Reduce the max read timeout to 1 second so we can test the timeout.
			MaxReadTimeout: 1,
			// Reduce the max connections to 3 so we can test the timeout.
			MaxConnections: 3,
			Port:           9418,
		},
	}
	ac, err := appCfg.NewConfig(cfg)
	if err != nil {
		log.Fatal(err)
	}
	d, err := NewDaemon(cfg, ac)
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
}

func TestMaxReadTimeout(t *testing.T) {
	c, err := net.Dial("tcp", testDaemon.addr)
	if err != nil {
		t.Fatal(err)
	}
	out, err := readPktline(c)
	if err != nil {
		t.Fatalf("expected nil, got error: %v", err)
	}
	if out != git.ErrMaxTimeout.Error() {
		t.Fatalf("expected %q error, got nil", git.ErrMaxTimeout)
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
		t.Fatalf("expected %q error, got nil", git.ErrInvalidRepo)
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
