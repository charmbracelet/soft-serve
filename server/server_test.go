package server

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/charmbracelet/keygen"
	"github.com/charmbracelet/soft-serve/config"
	"github.com/gliderlabs/ssh"
	"github.com/gogs/git-module"
	"github.com/matryer/is"
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
	is := is.New(t)
	_, pkPath = createKeyPair(t)
	s := setupServer(t)
	defer os.RemoveAll(testdata)
	err := s.Reload()
	is.NoErr(err)
	t.Run("TestPushRepo", testPushRepo)
	t.Run("TestCloneRepo", testCloneRepo)
}

func testPushRepo(t *testing.T) {
	is := is.New(t)
	rp := t.TempDir()
	defer os.RemoveAll(rp)
	err := git.Init(rp)
	is.NoErr(err)
	r, err := git.Open(rp)
	is.NoErr(err)
	tf := filepath.Join(rp, "testfile")
	f, err := os.Create(tf)
	is.NoErr(err)
	defer f.Close()
	err = r.Add(git.AddOptions{All: true})
	is.NoErr(err)
	err = git.CreateCommit(rp, &git.Signature{
		Name:  "test",
		Email: "",
	}, "test")
	is.NoErr(err)
	err = r.RemoteAdd("soft", fmt.Sprintf("ssh://%s:%d/test", cfg.Host, cfg.Port))
	is.NoErr(err)
	err = r.Push("soft", "master", git.PushOptions{
		Envs: []string{
			fmt.Sprintf("GIT_SSH_COMMAND=ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -i %s -F /dev/null", pkPath),
		},
	})
	is.NoErr(err)
}

func testCloneRepo(t *testing.T) {
	is := is.New(t)
	rp := t.TempDir()
	err := clone(fmt.Sprintf("ssh://%s:%d/config", cfg.Host, cfg.Port), rp, pkPath)
	is.NoErr(err)
}

func setupServer(t *testing.T) *Server {
	s := NewServer(cfg)
	go func() {
		s.Start()
	}()
	defer s.Close()
	return s
}

func createKeyPair(t *testing.T) (ssh.PublicKey, string) {
	is := is.New(t)
	t.Helper()

	keyDir := t.TempDir()
	_, err := keygen.NewWithWrite(keyDir, "id", nil, keygen.Ed25519)
	is.NoErr(err)
	pk := filepath.Join(keyDir, "id_ed25519")
	pubBytes, err := os.ReadFile(filepath.Join(keyDir, "id_ed25519.pub"))
	is.NoErr(err)
	pubkey, _, _, _, err := ssh.ParseAuthorizedKey(pubBytes)
	is.NoErr(err)
	return pubkey, pk
}

func clone(url, dst, pkPath string, opts ...git.CloneOptions) error {
	var opt git.CloneOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	err := os.MkdirAll(path.Dir(dst), os.ModePerm)
	if err != nil {
		return err
	}

	cmd := git.NewCommand("clone")
	if opt.Mirror {
		cmd.AddArgs("--mirror")
	}
	if opt.Bare {
		cmd.AddArgs("--bare")
	}
	if opt.Quiet {
		cmd.AddArgs("--quiet")
	}
	if !opt.Bare && opt.Branch != "" {
		cmd.AddArgs("-b", opt.Branch)
	}
	if opt.Depth > 0 {
		cmd.AddArgs("--depth", strconv.FormatUint(opt.Depth, 10))
	}
	cmd.AddEnvs(fmt.Sprintf("GIT_SSH_COMMAND=ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -i %s -F /dev/null", pkPath))

	_, err = cmd.AddArgs(url, dst).RunWithTimeout(opt.Timeout)
	return err
}
