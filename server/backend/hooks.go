package backend

import (
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
	PreReceive(stdout io.Writer, stderr io.Writer, repo string, args []HookArg)
	Update(stdout io.Writer, stderr io.Writer, repo string, arg HookArg)
	PostReceive(stdout io.Writer, stderr io.Writer, repo string, args []HookArg)
	PostUpdate(stdout io.Writer, stderr io.Writer, repo string, args ...string)
}
