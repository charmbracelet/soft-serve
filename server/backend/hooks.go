package backend

import (
	"context"
	"io"
	"sync"

	"github.com/charmbracelet/soft-serve/server/hooks"
	"github.com/charmbracelet/soft-serve/server/store"
)

var _ hooks.Hooks = (*Backend)(nil)

// PostReceive is called by the git post-receive hook.
//
// It implements Hooks.
func (d *Backend) PostReceive(ctx context.Context, stdout io.Writer, stderr io.Writer, repo string, args []hooks.HookArg) {
	d.logger.Debug("post-receive hook called", "repo", repo, "args", args)
}

// PreReceive is called by the git pre-receive hook.
//
// It implements Hooks.
func (d *Backend) PreReceive(ctx context.Context, stdout io.Writer, stderr io.Writer, repo string, args []hooks.HookArg) {
	d.logger.Debug("pre-receive hook called", "repo", repo, "args", args)
}

// Update is called by the git update hook.
//
// It implements Hooks.
func (d *Backend) Update(ctx context.Context, stdout io.Writer, stderr io.Writer, repo string, arg hooks.HookArg) {
	d.logger.Debug("update hook called", "repo", repo, "arg", arg)
}

// PostUpdate is called by the git post-update hook.
//
// It implements Hooks.
func (d *Backend) PostUpdate(ctx context.Context, stdout io.Writer, stderr io.Writer, repo string, args ...string) {
	d.logger.Debug("post-update hook called", "repo", repo, "args", args)

	var wg sync.WaitGroup

	// Populate last-modified file.
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := populateLastModified(ctx, d, repo); err != nil {
			d.logger.Error("error populating last-modified", "repo", repo, "err", err)
			return
		}
	}()

	wg.Wait()
}

func populateLastModified(ctx context.Context, d *Backend, name string) error {
	var rr *repo
	_rr, err := d.Repository(ctx, name)
	if err != nil {
		return err
	}

	if r, ok := _rr.(*repo); ok {
		rr = r
	} else {
		return store.ErrRepoNotExist
	}

	r, err := rr.Open()
	if err != nil {
		return err
	}

	c, err := r.LatestCommitTime()
	if err != nil {
		return err
	}

	return rr.writeLastModified(c)
}
