package server

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/keygen"
	ggit "github.com/charmbracelet/soft-serve/git"
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
	cfg = &config.Config{
		Host: "",
		Git:  config.GitConfig{Port: 9418},
	}
)

func TestPushRepo(t *testing.T) {
	is := is.New(t)
	_, pkPath := createKeyPair(t)
	s := setupServer(t)
	err := s.Reload()
	is.NoErr(err)
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
		HostKeyCallback: cssh.InsecureIgnoreHostKey(),
	}
	err = r.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       auth,
	})
	is.NoErr(err)
}

func TestCloneRepo(t *testing.T) {
	is := is.New(t)
	_, pkPath := createKeyPair(t)
	s := setupServer(t)
	log.Print("starting server")
	err := s.Reload()
	log.Print("reloaded server")
	is.NoErr(err)
	dst := t.TempDir()
	url := fmt.Sprintf("ssh://localhost:%d/config", cfg.SSH.Port)
	log.Print("cloning repo")
	err = ggit.Clone(url, dst, ggit.CloneOptions{
		CommandOptions: ggit.CommandOptions{
			Envs: []string{
				fmt.Sprintf(`GIT_SSH_COMMAND=ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -i %s -F /dev/null`, pkPath),
			},
		},
	})
	is.NoErr(err)
}

func randomPort() int {
	addr, _ := net.Listen("tcp", ":0") //nolint:gosec
	_ = addr.Close()
	return addr.Addr().(*net.TCPAddr).Port
}

func setupServer(t *testing.T) *Server {
	t.Helper()
	cfg.DataPath = t.TempDir()
	cfg.SSH.Port = randomPort()
	s := NewServer(cfg)
	go func() {
		log.Print("starting server")
		s.Start()
	}()
	t.Cleanup(func() {
		s.Close()
		os.RemoveAll(cfg.DataPath)
	})
	return s
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
