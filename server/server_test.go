package server

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/keygen"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/gliderlabs/ssh"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	gconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	gssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/matryer/is"
	gossh "golang.org/x/crypto/ssh"
)

func TestPushRepo(t *testing.T) {
	is := is.New(t)
	_, cfg, pkPath := setupServer(t)
	rp := t.TempDir()
	r, err := git.PlainInit(rp, false)
	is.NoErr(err)
	wt, err := r.Worktree()
	is.NoErr(err)
	_, err = wt.Filesystem.Create("testfile")
	is.NoErr(err)
	_, err = wt.Add("testfile")
	is.NoErr(err)
	author := &object.Signature{
		Name:  "test",
		Email: "",
	}
	_, err = wt.Commit("test commit", &git.CommitOptions{
		All:       true,
		Author:    author,
		Committer: author,
	})
	is.NoErr(err)
	_, err = r.CreateRemote(&gconfig.RemoteConfig{
		Name: "origin",
		URLs: []string{fmt.Sprintf("ssh://localhost:%d/%s", cfg.SSH.Port, "testrepo")},
	})
	auth, err := gssh.NewPublicKeysFromFile("git", pkPath, "")
	is.NoErr(err)
	auth.HostKeyCallbackHelper = gssh.HostKeyCallbackHelper{
		HostKeyCallback: gossh.InsecureIgnoreHostKey(),
	}
	t.Logf("pushing to ssh://localhost:%d/%s", cfg.SSH.Port, "testrepo")
	err = r.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       auth,
	})
	is.NoErr(err)
}

func TestCloneRepo(t *testing.T) {
	is := is.New(t)
	_, cfg, pkPath := setupServer(t)
	t.Log("starting server")
	dst := t.TempDir()
	t.Cleanup(func() { is.NoErr(os.RemoveAll(dst)) })
	url := fmt.Sprintf("ssh://localhost:%d/config", cfg.SSH.Port)
	t.Log("cloning repo")
	pk, err := gssh.NewPublicKeysFromFile("git", pkPath, "")
	is.NoErr(err)
	pk.HostKeyCallbackHelper = gssh.HostKeyCallbackHelper{
		HostKeyCallback: gossh.InsecureIgnoreHostKey(),
	}
	_, err = git.Clone(memory.NewStorage(), memfs.New(), &git.CloneOptions{
		URL:  url,
		Auth: pk,
	})
	is.NoErr(err)
}

func randomPort() int {
	addr, _ := net.Listen("tcp", ":0") //nolint:gosec
	_ = addr.Close()
	return addr.Addr().(*net.TCPAddr).Port
}

func setupServer(t *testing.T) (*Server, *config.Config, string) {
	t.Helper()
	is := is.New(t)
	pub, pkPath := createKeyPair(t)
	dp := t.TempDir()
	is.NoErr(os.Setenv("SOFT_SERVE_DATA_PATH", dp))
	is.NoErr(os.Setenv("SOFT_SERVE_INITIAL_ADMIN_KEY", authorizedKey(pub)))
	is.NoErr(os.Setenv("SOFT_SERVE_GIT_ENABLED", "false"))
	is.NoErr(os.Setenv("SOFT_SERVE_SSH_PORT", strconv.Itoa(randomPort())))
	// is.NoErr(os.Setenv("SOFT_SERVE_DB_DRIVER", "fake"))
	t.Cleanup(func() {
		is.NoErr(os.Unsetenv("SOFT_SERVE_DATA_PATH"))
		is.NoErr(os.Unsetenv("SOFT_SERVE_SSH_PORT"))
		is.NoErr(os.Unsetenv("SOFT_SERVE_INITIAL_ADMIN_KEY"))
		is.NoErr(os.Unsetenv("SOFT_SERVE_GIT_ENABLED"))
		// is.NoErr(os.Unsetenv("SOFT_SERVE_DB_DRIVER"))
		is.NoErr(os.RemoveAll(dp))
	})
	cfg := config.DefaultConfig() //.WithDB(&fakedb.FakeDB{})
	s := NewServer(cfg)
	go func() {
		t.Log("starting server")
		s.Start()
	}()
	t.Cleanup(func() {
		s.Close()
	})
	return s, cfg, pkPath
}

func createKeyPair(t *testing.T) (ssh.PublicKey, string) {
	t.Helper()
	is := is.New(t)
	keyDir := t.TempDir()
	kp, err := keygen.NewWithWrite(filepath.Join(keyDir, "id"), nil, keygen.Ed25519)
	is.NoErr(err)
	pubkey, _, _, _, err := ssh.ParseAuthorizedKey(kp.PublicKey())
	is.NoErr(err)
	return pubkey, filepath.Join(keyDir, "id_ed25519")
}

func authorizedKey(pk ssh.PublicKey) string {
	return strings.TrimSpace(string(gossh.MarshalAuthorizedKey(pk)))
}
