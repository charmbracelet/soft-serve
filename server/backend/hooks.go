package backend

import (
	"context"
	"io"
)

// HookArg is an argument to a git hook.
type HookArg struct {
	OldSha  string
	NewSha  string
	RefName string
}

// Hooks provides an interface for git server-side hooks.
type Hooks interface {
	PreReceive(ctx context.Context, stdout io.Writer, stderr io.Writer, repo string, args []HookArg)
	Update(ctx context.Context, stdout io.Writer, stderr io.Writer, repo string, arg HookArg)
	PostReceive(ctx context.Context, stdout io.Writer, stderr io.Writer, repo string, args []HookArg)
	PostUpdate(ctx context.Context, stdout io.Writer, stderr io.Writer, repo string, args ...string)
}

// PostReceive is called by the git post-receive hook.
//
// It implements Hooks.
func (d *Backend) PostReceive(ctx context.Context, stdout io.Writer, stderr io.Writer, repo string, args []HookArg) {
	d.logger.Debug("post-receive hook called", "repo", repo, "args", args)
}

// PreReceive is called by the git pre-receive hook.
//
// It implements Hooks.
func (d *Backend) PreReceive(ctx context.Context, stdout io.Writer, stderr io.Writer, repo string, args []HookArg) {
	d.logger.Debug("pre-receive hook called", "repo", repo, "args", args)
}

// Update is called by the git update hook.
//
// It implements Hooks.
func (d *Backend) Update(ctx context.Context, stdout io.Writer, stderr io.Writer, repo string, arg HookArg) {
	d.logger.Debug("update hook called", "repo", repo, "arg", arg)
}

// PostUpdate is called by the git post-update hook.
//
// It implements Hooks.
func (d *Backend) PostUpdate(ctx context.Context, stdout io.Writer, stderr io.Writer, repo string, args ...string) {
	d.logger.Debug("post-update hook called", "repo", repo, "args", args)

	if err := d.Touch(ctx, repo); err != nil {
		d.logger.Error("failed to touch repo", "repo", repo, "err", err)
	}
}
