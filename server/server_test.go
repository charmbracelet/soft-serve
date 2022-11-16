package server

import (
	"fmt"
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
		BindAddr: "",
		Host:     "localhost",
		Port:     22222,
	}
)

func TestPushRepo(t *testing.T) {
	is := is.New(t)
	_, pkPath := createKeyPair(t)
	s := setupServer(t)
	defer s.Close()
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

func TestCloneRepo(t *testing.T) {
	is := is.New(t)
	_, pkPath := createKeyPair(t)
	s := setupServer(t)
	defer s.Close()
	err := s.Reload()
	is.NoErr(err)

	dst := t.TempDir()
	url := fmt.Sprintf("ssh://%s:%d/config", cfg.Host, cfg.Port)
	err = ggit.Clone(url, dst, ggit.CloneOptions{
		CommandOptions: ggit.CommandOptions{
			Envs: []string{
				fmt.Sprintf(`GIT_SSH_COMMAND=ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -i %s -F /dev/null`, pkPath),
			},
		},
	})
	is.NoErr(err)
}

func setupServer(t *testing.T) *Server {
	t.Helper()
	tmpdir := t.TempDir()
	cfg.RepoPath = filepath.Join(tmpdir, "repos")
	cfg.KeyPath = filepath.Join(tmpdir, "key")
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
	t.Helper()
	is := is.New(t)
	keyDir := t.TempDir()
	kp, err := keygen.NewWithWrite(filepath.Join(keyDir, "id"), nil, keygen.Ed25519)
	is.NoErr(err)
	pubkey, _, _, _, err := ssh.ParseAuthorizedKey(kp.PublicKey())
	is.NoErr(err)
	return pubkey, filepath.Join(keyDir, "id_ed25519")
}
