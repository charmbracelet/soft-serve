package git

import "github.com/gogs/git-module"

// RunInDirOptions are options for RunInDir.
type RunInDirOptions = git.RunInDirOptions

// NewCommand creates a new git command.
func NewCommand(args ...string) *git.Command {
	return git.NewCommand(args...)
}
