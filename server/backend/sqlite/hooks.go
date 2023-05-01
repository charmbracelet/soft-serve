package sqlite

import (
	"io"
	"sync"

	"github.com/charmbracelet/soft-serve/server/backend"
)

// PostReceive is called by the git post-receive hook.
//
// It implements Hooks.
func (d *SqliteBackend) PostReceive(stdout io.Writer, stderr io.Writer, repo string, args []backend.HookArg) {
	d.logger.Debug("post-receive hook called", "repo", repo, "args", args)
}

// PreReceive is called by the git pre-receive hook.
//
// It implements Hooks.
func (d *SqliteBackend) PreReceive(stdout io.Writer, stderr io.Writer, repo string, args []backend.HookArg) {
	d.logger.Debug("pre-receive hook called", "repo", repo, "args", args)
}

// Update is called by the git update hook.
//
// It implements Hooks.
func (d *SqliteBackend) Update(stdout io.Writer, stderr io.Writer, repo string, arg backend.HookArg) {
	d.logger.Debug("update hook called", "repo", repo, "arg", arg)
}

// PostUpdate is called by the git post-update hook.
//
// It implements Hooks.
func (d *SqliteBackend) PostUpdate(stdout io.Writer, stderr io.Writer, repo string, args ...string) {
	d.logger.Debug("post-update hook called", "repo", repo, "args", args)

	var wg sync.WaitGroup

	// Update server info
	wg.Add(1)
	go func() {
		defer wg.Done()

		rr, err := d.Repository(repo)
		if err != nil {
			d.logger.Error("error getting repository", "repo", repo, "err", err)
			return
		}

		r, err := rr.Open()
		if err != nil {
			d.logger.Error("error opening repository", "repo", repo, "err", err)
			return
		}

		if err := r.UpdateServerInfo(); err != nil {
			d.logger.Error("error updating server-info", "repo", repo, "err", err)
			return
		}
	}()

	wg.Wait()
}
