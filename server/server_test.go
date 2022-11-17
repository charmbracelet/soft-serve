package server

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/keygen"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/gliderlabs/ssh"
	"github.com/go-git/go-git/v5"
	gconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	gssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/matryer/is"
	cssh "golang.org/x/crypto/ssh"
)

var (
	testdata = "testdata"
	cfg      = &config.Config{
		BindAddr: "",
		Host:     "localhost",
		Port:     22222,
		RepoPath: fmt.Sprintf("%s/repos", testdata),
		KeyPath:  fmt.Sprintf("%s/key", testdata),
	}
	pkPath = ""
)

func TestServer(t *testing.T) {
	t.Cleanup(func() {
		os.RemoveAll(testdata)
	})
	is := is.New(t)
	_, pkPath = createKeyPair(t)
	s := setupServer(t)
	err := s.Reload()
	is.NoErr(err)
	t.Run("TestPushRepo", testPushRepo)
	t.Run("TestCloneRepo", testCloneRepo)
}

func testPushRepo(t *testing.T) {
	is := is.New(t)
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
		URLs: []string{fmt.Sprintf("ssh://%s:%d/%s", cfg.Host, cfg.Port, "testrepo")},
	})
	auth, err := gssh.NewPublicKeysFromFile("git", pkPath, "")
	is.NoErr(err)
	auth.HostKeyCallbackHelper = gssh.HostKeyCallbackHelper{
		HostKeyCallback: cssh.InsecureIgnoreHostKey(),
	}
	err = r.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       auth,
	})
	is.NoErr(err)
}

func testCloneRepo(t *testing.T) {
	is := is.New(t)
	auth, err := gssh.NewPublicKeysFromFile("git", pkPath, "")
	is.NoErr(err)
	auth.HostKeyCallbackHelper = gssh.HostKeyCallbackHelper{
		HostKeyCallback: cssh.InsecureIgnoreHostKey(),
	}
	dst := t.TempDir()
	_, err = git.PlainClone(dst, false, &git.CloneOptions{
		URL:  fmt.Sprintf("ssh://%s:%d/config", cfg.Host, cfg.Port),
		Auth: auth,
	})
	is.NoErr(err)
}

func setupServer(t *testing.T) *Server {
	s := NewServer(cfg)
	go func() {
		s.Start()
	}()
	t.Cleanup(func() {
		s.Close()
	})
	return s
}

func createKeyPair(t *testing.T) (ssh.PublicKey, string) {
	is := is.New(t)
	t.Helper()
	keyDir := t.TempDir()
	kp, err := keygen.NewWithWrite(filepath.Join(keyDir, "id"), nil, keygen.Ed25519)
	is.NoErr(err)
	pubkey, _, _, _, err := ssh.ParseAuthorizedKey(kp.PublicKey())
	is.NoErr(err)
	return pubkey, filepath.Join(keyDir, "id_ed25519")
}
