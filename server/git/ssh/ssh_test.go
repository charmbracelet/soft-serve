package ssh

import (
	"fmt"
	"net"
	"net/mail"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/charmbracelet/keygen"
	"github.com/charmbracelet/soft-serve/proto"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	gossh "golang.org/x/crypto/ssh"
)

func TestGitMiddleware(t *testing.T) {
	pubkey, pkPath := createKeyPair(t)

	l, err := net.Listen("tcp", "127.0.0.1:0")
	requireNoError(t, err)
	remote := "ssh://" + l.Addr().String()

	repoDir := t.TempDir()
	hooks := &testHooks{
		access: []accessDetails{
			{pubkey, "repo1", proto.AdminAccess},
			{pubkey, "repo2", proto.AdminAccess},
			{pubkey, "repo3", proto.AdminAccess},
			{pubkey, "repo4", proto.AdminAccess},
			{pubkey, "repo5", proto.NoAccess},
			{pubkey, "repo6", proto.ReadOnlyAccess},
			{pubkey, "repo7", proto.AdminAccess},
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
	})

	t.Run("create repo on main", func(t *testing.T) {
		cwd := t.TempDir()
		requireNoError(t, runGitHelper(t, pkPath, cwd, "init", "-b", "main"))
		requireNoError(t, runGitHelper(t, pkPath, cwd, "remote", "add", "origin", remote+"/repo2"))
		requireNoError(t, runGitHelper(t, pkPath, cwd, "commit", "--allow-empty", "-m", "initial commit"))
		requireNoError(t, runGitHelper(t, pkPath, cwd, "push", "origin", "main"))
	})

	t.Run("create and clone repo", func(t *testing.T) {
		cwd := t.TempDir()
		requireNoError(t, runGitHelper(t, pkPath, cwd, "init", "-b", "main"))
		requireNoError(t, runGitHelper(t, pkPath, cwd, "remote", "add", "origin", remote+"/repo3"))
		requireNoError(t, runGitHelper(t, pkPath, cwd, "commit", "--allow-empty", "-m", "initial commit"))
		requireNoError(t, runGitHelper(t, pkPath, cwd, "push", "origin", "main"))

		cwd = t.TempDir()
		requireNoError(t, runGitHelper(t, pkPath, cwd, "clone", remote+"/repo3"))
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
	cmd.Env = []string{fmt.Sprintf(`GIT_SSH_COMMAND=ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -i "%s" -F /dev/null`, pk)}
	out, err := cmd.CombinedOutput()
	t.Log("git out:", string(out))
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
		t.Logf("action: %q", action.repo)
		if repo == strings.TrimSuffix(action.repo, ".git") && ssh.KeysEqual(key, action.key) {
			return
		}
	}
	t.Fatalf("expected action for %q, got none", repo)
}

func createKeyPair(t *testing.T) (ssh.PublicKey, string) {
	t.Helper()

	keyDir := t.TempDir()
	t.Logf("Tempdir %s", keyDir)
	kp, err := keygen.NewWithWrite(filepath.Join(keyDir, "id"), nil, keygen.Ed25519)
	kp.KeyPairExists()
	requireNoError(t, err)
	pk := filepath.Join(keyDir, "id_ed25519")
	t.Logf("pk %s", pk)
	pubBytes, err := os.ReadFile(filepath.Join(keyDir, "id_ed25519.pub"))
	requireNoError(t, err)
	pubkey, _, _, _, err := ssh.ParseAuthorizedKey(pubBytes)
	requireNoError(t, err)
	return pubkey, pk
}

type accessDetails struct {
	key   ssh.PublicKey
	repo  string
	level proto.AccessLevel
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

func (h *testHooks) Open(string) (proto.Repository, error) {
	return nil, nil
}

func (h *testHooks) ListRepos() ([]proto.Metadata, error) {
	return nil, nil
}

func (h *testHooks) AuthRepo(repo string, key ssh.PublicKey) proto.AccessLevel {
	for _, dets := range h.access {
		if strings.TrimSuffix(dets.repo, ".git") == repo && ssh.KeysEqual(key, dets.key) {
			return dets.level
		}
	}
	return proto.NoAccess
}

type testUser struct{}

func (u *testUser) Name() string {
	return "test"
}

func (u *testUser) Email() *mail.Address {
	return &mail.Address{
		Name:    "test",
		Address: "test@wish",
	}
}

func (u *testUser) IsAdmin() bool {
	return false
}

func (u *testUser) Login() *string {
	l := "test"
	return &l
}

func (u *testUser) Password() *string {
	return nil
}

func (u *testUser) PublicKeys() []gossh.PublicKey {
	return nil
}

func (h *testHooks) User(pk ssh.PublicKey) (proto.User, error) {
	return &testUser{}, nil
}

func (h *testHooks) IsAdmin(pk ssh.PublicKey) bool {
	return false
}

func (h *testHooks) IsCollab(repo string, pk ssh.PublicKey) bool {
	return false
}

func (h *testHooks) Create(name, projectName, description string, isPrivate bool) error {
	return nil
}

func (h *testHooks) Delete(repo string) error {
	return nil
}

func (h *testHooks) Rename(repo, name string) error {
	return nil
}

func (h *testHooks) SetProjectName(repo, projectName string) error {
	return nil
}

func (h *testHooks) SetDescription(repo, description string) error {
	return nil
}

func (h *testHooks) SetPrivate(repo string, isPrivate bool) error {
	return nil
}

func (h *testHooks) SetDefaultBranch(repo, branch string) error {
	return nil
}

func (h *testHooks) Push(repo string, key ssh.PublicKey) {
	h.Lock()
	defer h.Unlock()

	h.pushes = append(h.pushes, action{key, strings.TrimSuffix(repo, ".git")})
}

func (h *testHooks) Fetch(repo string, key ssh.PublicKey) {
	h.Lock()
	defer h.Unlock()

	h.fetches = append(h.fetches, action{key, repo})
}
