package hooks

import "io"

// HookArg is an argument to a git hook.
type HookArg struct {
	OldSha  string
	NewSha  string
	RefName string
}

// Hooks provides an interface for git server-side hooks.
type Hooks interface {
	PreReceive(stdin io.Reader, stdout io.Writer, stderr io.Writer, repo string, args []HookArg)
	Update(stdin io.Reader, stdout io.Writer, stderr io.Writer, repo string, arg HookArg)
	PostReceive(stdin io.Reader, stdout io.Writer, stderr io.Writer, repo string, args []HookArg)
	PostUpdate(stdin io.Reader, stdout io.Writer, stderr io.Writer, repo string, args ...string)
}
