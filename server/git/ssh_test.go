package git

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/charmbracelet/keygen"
	"github.com/charmbracelet/wish"
	"github.com/gliderlabs/ssh"
)

func TestGitMiddleware(t *testing.T) {
	pubkey, pkPath := createKeyPair(t)

	l, err := net.Listen("tcp", "127.0.0.1:0")
	requireNoError(t, err)
	remote := "ssh://" + l.Addr().String()

	repoDir := t.TempDir()
	hooks := &testHooks{
		pushes:  []action{},
		fetches: []action{},
		access: []accessDetails{
			{pubkey, "repo1", AdminAccess},
			{pubkey, "repo2", AdminAccess},
			{pubkey, "repo3", AdminAccess},
			{pubkey, "repo4", AdminAccess},
			{pubkey, "repo5", NoAccess},
			{pubkey, "repo6", ReadOnlyAccess},
			{pubkey, "repo7", AdminAccess},
		},
	}
	srv, err := wish.NewServer(
		wish.WithMiddleware(Middleware(repoDir, hooks)),
		wish.WithPublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
			return true
		}),
	)
	requireNoError(t, err)
	go func() { srv.Serve(l) }()
	t.Cleanup(func() { _ = srv.Close() })

	t.Run("create repo on master", func(t *testing.T) {
		cwd := t.TempDir()
		requireNoError(t, runGitHelper(t, pkPath, cwd, "init", "-b", "master"))
		requireNoError(t, runGitHelper(t, pkPath, cwd, "remote", "add", "origin", remote+"/repo1"))
		requireNoError(t, runGitHelper(t, pkPath, cwd, "commit", "--allow-empty", "-m", "initial commit"))
		requireNoError(t, runGitHelper(t, pkPath, cwd, "push", "origin", "master"))
		requireHasAction(t, hooks.pushes, pubkey, "repo1")
	})

	t.Run("create repo on main", func(t *testing.T) {
		cwd := t.TempDir()
		requireNoError(t, runGitHelper(t, pkPath, cwd, "init", "-b", "main"))
		requireNoError(t, runGitHelper(t, pkPath, cwd, "remote", "add", "origin", remote+"/repo2"))
		requireNoError(t, runGitHelper(t, pkPath, cwd, "commit", "--allow-empty", "-m", "initial commit"))
		requireNoError(t, runGitHelper(t, pkPath, cwd, "push", "origin", "main"))
		requireHasAction(t, hooks.pushes, pubkey, "repo2")
	})

	t.Run("create and clone repo", func(t *testing.T) {
		cwd := t.TempDir()
		requireNoError(t, runGitHelper(t, pkPath, cwd, "init", "-b", "main"))
		requireNoError(t, runGitHelper(t, pkPath, cwd, "remote", "add", "origin", remote+"/repo3"))
		requireNoError(t, runGitHelper(t, pkPath, cwd, "commit", "--allow-empty", "-m", "initial commit"))
		requireNoError(t, runGitHelper(t, pkPath, cwd, "push", "origin", "main"))

		cwd = t.TempDir()
		requireNoError(t, runGitHelper(t, pkPath, cwd, "clone", remote+"/repo3"))

		requireHasAction(t, hooks.pushes, pubkey, "repo3")
		requireHasAction(t, hooks.fetches, pubkey, "repo3")
	})

	t.Run("clone repo that doesn't exist", func(t *testing.T) {
		cwd := t.TempDir()
		requireError(t, runGitHelper(t, pkPath, cwd, "clone", remote+"/repo4"))
	})

	t.Run("clone repo with no access", func(t *testing.T) {
		cwd := t.TempDir()
		requireError(t, runGitHelper(t, pkPath, cwd, "clone", remote+"/repo5"))
	})

	t.Run("push repo with with readonly", func(t *testing.T) {
		cwd := t.TempDir()
		requireNoError(t, runGitHelper(t, pkPath, cwd, "init", "-b", "main"))
		requireNoError(t, runGitHelper(t, pkPath, cwd, "remote", "add", "origin", remote+"/repo6"))
		requireNoError(t, runGitHelper(t, pkPath, cwd, "commit", "--allow-empty", "-m", "initial commit"))
		requireError(t, runGitHelper(t, pkPath, cwd, "push", "origin", "main"))
	})

	t.Run("create and clone repo on weird branch", func(t *testing.T) {
		cwd := t.TempDir()
		requireNoError(t, runGitHelper(t, pkPath, cwd, "init", "-b", "a-weird-branch-name"))
		requireNoError(t, runGitHelper(t, pkPath, cwd, "remote", "add", "origin", remote+"/repo7"))
		requireNoError(t, runGitHelper(t, pkPath, cwd, "commit", "--allow-empty", "-m", "initial commit"))
		requireNoError(t, runGitHelper(t, pkPath, cwd, "push", "origin", "a-weird-branch-name"))

		cwd = t.TempDir()
		requireNoError(t, runGitHelper(t, pkPath, cwd, "clone", remote+"/repo7"))

		requireHasAction(t, hooks.pushes, pubkey, "repo7")
		requireHasAction(t, hooks.fetches, pubkey, "repo7")
	})
}

func runGitHelper(t *testing.T, pk, cwd string, args ...string) error {
	t.Helper()

	allArgs := []string{
		"-c", "user.name='wish'",
		"-c", "user.email='test@wish'",
		"-c", "commit.gpgSign=false",
		"-c", "tag.gpgSign=false",
		"-c", "log.showSignature=false",
		"-c", "ssh.variant=ssh",
	}
	allArgs = append(allArgs, args...)

	cmd := exec.Command("git", allArgs...)
	cmd.Dir = cwd
	cmd.Env = []string{fmt.Sprintf(`GIT_SSH_COMMAND=ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -i %s -F /dev/null`, pk)}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Log("git out:", string(out))
	}
	return err
}

func requireNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("expected no error, got %q", err.Error())
	}
}

func requireError(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected an error, got nil")
	}
}

func requireHasAction(t *testing.T, actions []action, key ssh.PublicKey, repo string) {
	t.Helper()

	for _, action := range actions {
		r1 := repo
		if !strings.HasSuffix(r1, ".git") {
			r1 += ".git"
		}
		r2 := action.repo
		if !strings.HasSuffix(r2, ".git") {
			r2 += ".git"
		}
		if r1 == r2 && ssh.KeysEqual(key, action.key) {
			return
		}
	}
	t.Fatalf("expected action for %q, got none", repo)
}

func createKeyPair(t *testing.T) (ssh.PublicKey, string) {
	t.Helper()

	keyDir := t.TempDir()
	_, err := keygen.NewWithWrite(filepath.Join(keyDir, "id"), nil, keygen.Ed25519)
	requireNoError(t, err)
	pk := filepath.Join(keyDir, "id_ed25519")
	pubBytes, err := os.ReadFile(filepath.Join(keyDir, "id_ed25519.pub"))
	requireNoError(t, err)
	pubkey, _, _, _, err := ssh.ParseAuthorizedKey(pubBytes)
	requireNoError(t, err)
	return pubkey, pk
}

type accessDetails struct {
	key   ssh.PublicKey
	repo  string
	level AccessLevel
}

type action struct {
	key  ssh.PublicKey
	repo string
}

type testHooks struct {
	sync.Mutex
	pushes  []action
	fetches []action
	access  []accessDetails
}

func (h *testHooks) AuthRepo(repo string, key ssh.PublicKey) AccessLevel {
	for _, dets := range h.access {
		r1 := strings.TrimSuffix(dets.repo, ".git")
		r2 := strings.TrimSuffix(repo, ".git")
		if r1 == r2 && ssh.KeysEqual(key, dets.key) {
			return dets.level
		}
	}
	return NoAccess
}

func (h *testHooks) Push(repo string, key ssh.PublicKey) {
	h.Lock()
	defer h.Unlock()

	h.pushes = append(h.pushes, action{key, repo})
}

func (h *testHooks) Fetch(repo string, key ssh.PublicKey) {
	h.Lock()
	defer h.Unlock()

	h.fetches = append(h.fetches, action{key, repo})
}
