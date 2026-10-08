package backend

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/matryer/is"
)

// TestImportRepositoryCloneFailure verifies that ImportRepository returns the
// clone error instead of blocking forever when the remote cannot be cloned.
func TestImportRepositoryCloneFailure(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found")
	}
	if _, err := exec.LookPath("ssh"); err != nil {
		t.Skip("ssh not found")
	}

	is := is.New(t)
	be, _ := newTestBackend(t)
	ctx := context.Background()

	// Nothing listens on port 1, so the clone fails right away.
	remote := "ssh://git@127.0.0.1:1/missing.git"

	type result struct {
		repo proto.Repository
		err  error
	}
	resc := make(chan result, 1)
	go func() {
		r, err := be.ImportRepository(ctx, "import-fail", nil, remote, proto.RepositoryOptions{})
		resc <- result{r, err}
	}()

	select {
	case res := <-resc:
		is.True(res.err != nil) // a failed clone is reported
		is.True(res.repo == nil)
	case <-time.After(30 * time.Second):
		t.Fatal("ImportRepository did not return after the clone failed")
	}

	_, err := be.Repository(ctx, "import-fail")
	is.True(err != nil) // no repository is left behind
}
