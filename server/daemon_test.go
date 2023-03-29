package server

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/soft-serve/server/backend/file"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/go-git/go-git/v5/plumbing/format/pktline"
)

var testDaemon *GitDaemon

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "soft-serve-test")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	os.Setenv("SOFT_SERVE_DATA_PATH", tmp)
	os.Setenv("SOFT_SERVE_GIT_MAX_CONNECTIONS", "3")
	os.Setenv("SOFT_SERVE_GIT_MAX_TIMEOUT", "100")
	os.Setenv("SOFT_SERVE_GIT_IDLE_TIMEOUT", "1")
	os.Setenv("SOFT_SERVE_GIT_LISTEN_ADDR", fmt.Sprintf(":%d", randomPort()))
	fb, err := file.NewFileBackend(filepath.Join(tmp, "repos"))
	if err != nil {
		log.Fatal(err)
	}
	cfg := config.DefaultConfig().WithBackend(fb)
	d, err := NewGitDaemon(cfg)
	if err != nil {
		log.Fatal(err)
	}
	testDaemon = d
	go func() {
		if err := d.Start(); err != ErrServerClosed {
			log.Fatal(err)
		}
	}()
	code := m.Run()
	os.Unsetenv("SOFT_SERVE_DATA_PATH")
	os.Unsetenv("SOFT_SERVE_GIT_MAX_CONNECTIONS")
	os.Unsetenv("SOFT_SERVE_GIT_MAX_TIMEOUT")
	os.Unsetenv("SOFT_SERVE_GIT_IDLE_TIMEOUT")
	os.Unsetenv("SOFT_SERVE_GIT_LISTEN_ADDR")
	_ = d.Close()
	os.Exit(code)
}

func TestIdleTimeout(t *testing.T) {
	c, err := net.Dial("tcp", testDaemon.addr)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Second)
	out, err := readPktline(c)
	if err != nil && !errors.Is(err, io.EOF) {
		t.Fatalf("expected nil, got error: %v", err)
	}
	if out != ErrTimeout.Error() {
		t.Fatalf("expected %q error, got %q", ErrTimeout, out)
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
	if out != ErrInvalidRepo.Error() {
		t.Fatalf("expected %q error, got %q", ErrInvalidRepo, out)
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
	return strings.TrimSpace(string(pktout.Bytes())), nil
}
