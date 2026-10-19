package backend

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"slices"
	"testing"
	"time"

	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/ssrf"
	"github.com/matryer/is"
)

// TestCreateRepositoryAnonymousOwner verifies that a repository created with
// a nil user (an anon-access/allow-keyless override) is owned by the
// lowest-ID admin rather than failing the NOT NULL repos.user_id constraint.
func TestCreateRepositoryAnonymousOwner(t *testing.T) {
	is := is.New(t)
	be, _ := newTestBackend(t)
	ctx := context.Background()

	admin, err := be.User(ctx, "admin")
	is.NoErr(err)

	repo, err := be.CreateRepository(ctx, "anon-repo", nil, proto.RepositoryOptions{})
	is.NoErr(err)
	is.Equal(repo.UserID(), admin.ID())
}

// TestDefaultAdminUserIDNoAdmin verifies that defaultAdminUserID surfaces an
// error rather than silently returning a zero user ID (which would violate
// the NOT NULL repos.user_id constraint) when the database has no admin.
func TestDefaultAdminUserIDNoAdmin(t *testing.T) {
	is := is.New(t)
	be, _ := newTestBackend(t)
	ctx := context.Background()

	err := be.db.TransactionContext(ctx, func(tx *db.Tx) error {
		if _, err := tx.ExecContext(ctx, "UPDATE users SET admin = false"); err != nil {
			return err
		}

		_, err := be.defaultAdminUserID(ctx, tx)
		return err
	})
	is.True(err != nil)
}

// TestValidateImportRemote verifies that import remotes pointing at private,
// internal, or non-network destinations are rejected, that the ssh policy
// maps to the right user-facing error, and that accepted remotes carry the
// git environment that keeps the validation honest.
func TestValidateImportRemote(t *testing.T) {
	allowed, err := ssrf.ParseAllowedHosts([]string{"git.internal.example", "10.0.0.0/8"})
	if err != nil {
		t.Fatal(err)
	}
	nobody := ssrf.GitRemotePolicy{}
	admin := ssrf.GitRemotePolicy{AllowAnySSHHost: true}
	listed := ssrf.GitRemotePolicy{AllowedSSHHosts: allowed}

	tests := []struct {
		name    string
		remote  string
		policy  ssrf.GitRemotePolicy
		wantErr error
	}{
		{"public https", "https://1.1.1.1/x.git", nobody, nil},
		{"public git", "git://1.1.1.1/x.git", nobody, nil},
		// With no allowlist, ssh is for admins only; the user sees an
		// authorization error, not a remote one.
		{"ssh admin", "ssh://git@10.0.0.1/x.git", admin, nil},
		{"ssh user", "ssh://git@10.0.0.1/x.git", nobody, proto.ErrUnauthorized},
		{"ssh allowed by name", "ssh://git@git.internal.example/x.git", listed, nil},
		{"ssh allowed by cidr", "git@10.1.1.1:x.git", listed, nil},
		{"ssh not on allowlist", "ssh://git@evil.example/x.git", listed, proto.ErrInvalidRemote},
		{"scp no user", "github.com:charmbracelet/soft-serve.git", admin, nil},
		{"scp ipv6", "git@[::1]:x.git", admin, nil},

		{"loopback", "http://127.0.0.1/x.git", nobody, proto.ErrInvalidRemote},
		{"localhost", "http://localhost:8080/x.git", nobody, proto.ErrInvalidRemote},
		{"private network", "http://10.0.0.1/x.git", nobody, proto.ErrInvalidRemote},
		{"cloud metadata", "http://169.254.169.254/latest/meta-data/", nobody, proto.ErrInvalidRemote},
		{"git scheme private", "git://10.0.0.1/x.git", nobody, proto.ErrInvalidRemote},
		{"octal loopback", "http://0177.0.0.1/x.git", nobody, proto.ErrInvalidRemote},
		{"local path", "/data/repos/secret.git", nobody, proto.ErrInvalidRemote},
		{"file scheme", "file:///etc/passwd", nobody, proto.ErrInvalidRemote},
		{"empty", "", nobody, proto.ErrInvalidRemote},
		// Regression: a URL Go cannot parse but curl can was classified as
		// SSH and skipped validation entirely.
		{"unparseable http to loopback", "http://a{b@127.0.0.1:8080/", nobody, proto.ErrInvalidRemote},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			v, err := validateImportRemote(tt.remote, tt.policy)

			if tt.wantErr != nil {
				is.True(errors.Is(err, tt.wantErr))
				return
			}

			is.NoErr(err)
			env := ssrf.GitEnv(v)
			// Redirect following must be off, or a public remote can hand
			// off to an internal one after validation has passed.
			is.True(slices.Contains(env, "GIT_CONFIG_VALUE_0=false"))
			is.True(slices.Contains(env, "GIT_CONFIG_KEY_0=http.followRedirects"))
		})
	}
}

// TestImportRepositoryCloneFailure verifies that a failed clone returns to the
// caller promptly with a fixed error and leaves nothing behind. Before the fix
// the caller blocked forever on a channel that was only written after a
// successful clone, leaking a goroutine per failed import.
func TestImportRepositoryCloneFailure(t *testing.T) {
	if _, err := exec.LookPath("ssh"); err != nil {
		t.Skip("ssh client not available")
	}

	is := is.New(t)
	be, _ := newTestBackend(t)
	ctx := context.Background()

	admin, err := be.User(ctx, "admin")
	is.NoErr(err)

	// An ssh remote passes validation, and nothing listens on port 1 of
	// loopback, so the clone is refused immediately without touching the
	// network.
	type result struct {
		repo proto.Repository
		err  error
	}
	resc := make(chan result, 1)
	go func() {
		r, err := be.ImportRepository(ctx, "broken", admin, "ssh://127.0.0.1:1/x.git", proto.RepositoryOptions{})
		resc <- result{r, err}
	}()

	select {
	case res := <-resc:
		is.True(res.repo == nil)
		is.True(errors.Is(res.err, proto.ErrImportFailed))
		// The clone's own message must not reach the caller.
		is.Equal(res.err.Error(), proto.ErrImportFailed.Error())
	case <-time.After(30 * time.Second):
		t.Fatal("ImportRepository did not return after a failed clone")
	}

	_, err = be.Repository(ctx, "broken")
	is.True(errors.Is(err, proto.ErrRepoNotFound))
	_, err = os.Stat(be.repoPath("broken"))
	is.True(errors.Is(err, os.ErrNotExist))
}

func TestRedactRemote(t *testing.T) {
	tests := map[string]string{
		"https://user:secret@example.com/x.git": "https://user:xxxxx@example.com/x.git",
		"https://user@example.com/x.git":        "https://user@example.com/x.git",
		"https://example.com/x.git":             "https://example.com/x.git",
		"git@github.com:x/y.git":                "git@github.com:x/y.git",
		"ssh://git@10.0.0.1/x.git":              "ssh://git@10.0.0.1/x.git",
	}
	for in, want := range tests {
		if got := redactRemote(in); got != want {
			t.Errorf("redactRemote(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestImportRepositorySSHRequiresAdmin verifies that with no ssh allowlist
// configured, only a server admin may import over ssh. The clone would
// present the server's client key to whatever host the remote names.
func TestImportRepositorySSHRequiresAdmin(t *testing.T) {
	is := is.New(t)
	be, cfg := newTestBackend(t)
	ctx := context.Background()
	is.Equal(len(cfg.SSH.AllowedHosts()), 0)

	user, err := be.CreateUser(ctx, "tenant", proto.UserOptions{})
	is.NoErr(err)

	for _, remote := range []string{
		"ssh://git@10.0.0.1/x.git",
		"git+ssh://git@10.0.0.1/x.git",
		"git@10.0.0.1:x.git",
		"[git@10.0.0.1:2222]:x.git",
	} {
		_, err = be.ImportRepository(ctx, "tenant-ssh", user, remote, proto.RepositoryOptions{})
		is.True(errors.Is(err, proto.ErrUnauthorized)) // remote
	}

	_, err = be.ImportRepository(ctx, "tenant-ssh", nil, "ssh://git@10.0.0.1/x.git", proto.RepositoryOptions{})
	is.True(errors.Is(err, proto.ErrUnauthorized))

	// An ssh LFS endpoint is subject to the same policy as the remote.
	_, err = be.ImportRepository(ctx, "tenant-ssh", user, "https://1.1.1.1/x.git", proto.RepositoryOptions{LFSEndpoint: "ssh://git@10.0.0.1/x.git"})
	is.True(errors.Is(err, proto.ErrUnauthorized))

	// An allowlist entry that does not cover the host is refused for
	// everyone, admin included, with the remote error rather than an
	// authorization one.
	admin, err := be.User(ctx, "admin")
	is.NoErr(err)
	cfg.SSH.ClientAllowedHosts = []string{"git.internal.example"}
	is.NoErr(cfg.Validate())
	_, err = be.ImportRepository(ctx, "admin-ssh", admin, "ssh://git@10.0.0.1/x.git", proto.RepositoryOptions{})
	is.True(errors.Is(err, proto.ErrInvalidRemote))
	is.True(errors.Is(err, ssrf.ErrSSHHostNotAllowed))
	_, err = be.ImportRepository(ctx, "tenant-ssh", user, "ssh://git@10.0.0.1/x.git", proto.RepositoryOptions{})
	is.True(errors.Is(err, proto.ErrInvalidRemote))

	// Nothing was created along the way.
	_, err = be.Repository(ctx, "tenant-ssh")
	is.True(errors.Is(err, proto.ErrRepoNotFound))
	_, err = be.Repository(ctx, "admin-ssh")
	is.True(errors.Is(err, proto.ErrRepoNotFound))
}

// TestImportRepositoryDisabled verifies that when the installed git is too
// old to honour the remote guards, or its version cannot be read, imports are
// refused before any remote is touched, and that a new enough git leaves the
// gate open.
func TestImportRepositoryDisabled(t *testing.T) {
	tests := []struct {
		name     string
		version  git.Version
		err      error
		disabled bool
	}{
		{"too old", git.Version{Major: 2, Minor: 30, Patch: 2}, nil, true},
		{"just below", git.Version{Major: 2, Minor: 32, Patch: 9}, nil, true},
		{"minimum", git.Version{Major: 2, Minor: 33}, nil, false},
		{"current", git.Version{Major: 2, Minor: 45}, nil, false},
		{"unreadable", git.Version{}, errors.New("exec: git: not found"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			orig := binVersion
			binVersion = func() (git.Version, error) { return tt.version, tt.err }
			t.Cleanup(func() { binVersion = orig })

			be, _ := newTestBackend(t)
			ctx := context.Background()
			admin, err := be.User(ctx, "admin")
			is.NoErr(err)

			reason := be.ImportsDisabledReason()
			is.Equal(reason != "", tt.disabled)
			if !tt.disabled {
				return
			}

			_, err = be.ImportRepository(ctx, "gated", admin, "https://1.1.1.1/x.git", proto.RepositoryOptions{})
			is.True(errors.Is(err, proto.ErrImportsDisabled))
			// The reason is for the operator, not the user.
			is.Equal(err.Error(), proto.ErrImportsDisabled.Error())
			_, err = be.Repository(ctx, "gated")
			is.True(errors.Is(err, proto.ErrRepoNotFound))
		})
	}
}
