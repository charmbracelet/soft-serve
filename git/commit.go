package git

import (
	"github.com/gogs/git-module"
)

// ZeroID is the zero hash.
const ZeroID = git.EmptyID

// Commit is a wrapper around git.Commit with helper methods.
type Commit = git.Commit

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
