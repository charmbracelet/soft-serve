package server

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/keygen"
	"github.com/charmbracelet/soft-serve/server/test"
	"github.com/charmbracelet/ssh"
	"github.com/matryer/is"
	gossh "golang.org/x/crypto/ssh"
)

func setupServer(tb testing.TB) (*Server, string) {
	tb.Helper()
	tb.Log("creating keypair")
	pub, pkPath := createKeyPair(tb)
	dp := tb.TempDir()
	sshPort := fmt.Sprintf(":%d", test.RandomPort())
	tb.Setenv("SOFT_SERVE_DATA_PATH", dp)
	tb.Setenv("SOFT_SERVE_INITIAL_ADMIN_KEY", authorizedKey(pub))
	tb.Setenv("SOFT_SERVE_SSH_LISTEN_ADDR", sshPort)
	tb.Setenv("SOFT_SERVE_GIT_LISTEN_ADDR", fmt.Sprintf(":%d", test.RandomPort()))
	ctx := context.TODO()
	tb.Log("configuring server")
	s, err := NewServer(ctx)
	if err != nil {
		tb.Fatal(err)
	}
	go func() {
		tb.Log("starting server")
		s.Start()
	}()
	tb.Cleanup(func() {
		s.Close()
	})
	return s, pkPath
}

func createKeyPair(tb testing.TB) (ssh.PublicKey, string) {
	tb.Helper()
	is := is.New(tb)
	keyDir := tb.TempDir()
	fp := filepath.Join(keyDir, "id_ed25519")
	kp, err := keygen.New(fp, keygen.WithKeyType(keygen.Ed25519), keygen.WithWrite())
	is.NoErr(err)
	return kp.PublicKey(), fp
}

func authorizedKey(pk ssh.PublicKey) string {
	return strings.TrimSpace(string(gossh.MarshalAuthorizedKey(pk)))
}
