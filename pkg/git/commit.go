package git

import (
	"github.com/gogs/git-module"
)

var (
	ZeroHash Hash = git.EmptyID
)

// Hash represents a git hash.
type Hash string

// String returns the string representation of a hash as a string.
func (h Hash) String() string {
	return string(h)
}

// SHA1 represents the hash as a SHA1.
func (h Hash) SHA1() *git.SHA1 {
	return git.MustIDFromString(h.String())
}

// Commit is a wrapper around git.Commit with helper methods.
type Commit struct {
	*git.Commit
	Hash Hash
}

// Commits is a list of commits.
type Commits []*Commit

// Len implements sort.Interface.
func (cl Commits) Len() int { return len(cl) }

// Swap implements sort.Interface.
func (cl Commits) Swap(i, j int) { cl[i], cl[j] = cl[j], cl[i] }

// Less implements sort.Interface.
func (cl Commits) Less(i, j int) bool {
	return cl[i].Author.When.After(cl[j].Author.When)
}
