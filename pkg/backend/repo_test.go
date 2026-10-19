package backend

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"slices"
	"testing"
	"time"

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
// internal, or non-network destinations are rejected, and that accepted
// remotes carry the git environment that keeps the validation honest.
func TestValidateImportRemote(t *testing.T) {
	tests := []struct {
		name    string
		remote  string
		wantErr bool
	}{
		{"public https", "https://1.1.1.1/x.git", false},
		{"public git", "git://1.1.1.1/x.git", false},
		{"ssh", "ssh://git@10.0.0.1/x.git", false},
		{"scp no user", "github.com:charmbracelet/soft-serve.git", false},
		{"scp ipv6", "git@[::1]:x.git", false},

		{"loopback", "http://127.0.0.1/x.git", true},
		{"localhost", "http://localhost:8080/x.git", true},
		{"private network", "http://10.0.0.1/x.git", true},
		{"cloud metadata", "http://169.254.169.254/latest/meta-data/", true},
		{"git scheme private", "git://10.0.0.1/x.git", true},
		{"octal loopback", "http://0177.0.0.1/x.git", true},
		{"local path", "/data/repos/secret.git", true},
		{"file scheme", "file:///etc/passwd", true},
		{"empty", "", true},
		// Regression: a URL Go cannot parse but curl can was classified as
		// SSH and skipped validation entirely.
		{"unparseable http to loopback", "http://a{b@127.0.0.1:8080/", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			v, err := validateImportRemote(tt.remote)

			if tt.wantErr {
				is.True(err != nil)
				is.True(errors.Is(err, proto.ErrInvalidRemote))
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
