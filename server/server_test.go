package server

import (
	"context"
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/keygen"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/ssh"
	"github.com/matryer/is"
	gossh "golang.org/x/crypto/ssh"
)

func randomPort() int {
	addr, _ := net.Listen("tcp", ":0") //nolint:gosec
	_ = addr.Close()
	return addr.Addr().(*net.TCPAddr).Port
}

func setupServer(tb testing.TB) (*Server, *config.Config, string) {
	tb.Helper()
	tb.Log("creating keypair")
	pub, pkPath := createKeyPair(tb)
	dp := tb.TempDir()
	sshPort := fmt.Sprintf(":%d", randomPort())
	tb.Setenv("SOFT_SERVE_DATA_PATH", dp)
	tb.Setenv("SOFT_SERVE_INITIAL_ADMIN_KEY", authorizedKey(pub))
	tb.Setenv("SOFT_SERVE_SSH_LISTEN_ADDR", sshPort)
	tb.Setenv("SOFT_SERVE_GIT_LISTEN_ADDR", fmt.Sprintf(":%d", randomPort()))
	cfg := config.DefaultConfig()
	tb.Log("configuring server")
	s, err := NewServer(cfg)
	if err != nil {
		tb.Fatal(err)
	}
	go func() {
		tb.Log("starting server")
		s.Start(context.TODO())
	}()
	tb.Cleanup(func() {
		s.Close()
	})
	return s, cfg, pkPath
}

func createKeyPair(tb testing.TB) (ssh.PublicKey, string) {
	tb.Helper()
	is := is.New(tb)
	keyDir := tb.TempDir()
	kp, err := keygen.NewWithWrite(filepath.Join(keyDir, "id"), nil, keygen.Ed25519)
	is.NoErr(err)
	pubkey, _, _, _, err := ssh.ParseAuthorizedKey(kp.PublicKey())
	is.NoErr(err)
	return pubkey, filepath.Join(keyDir, "id_ed25519")
}

func authorizedKey(pk ssh.PublicKey) string {
	return strings.TrimSpace(string(gossh.MarshalAuthorizedKey(pk)))
}
