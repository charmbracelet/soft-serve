package git

import (
	"regexp"

	"github.com/aymanbagabas/git-module"
)

// ZeroID is the zero hash.
const ZeroID = git.EmptyID

// zeroHashPattern matches an all-zero SHA-1 (40 hex zeros) or SHA-256
// (64 hex zeros) object ID. The alternation prevents matching lengths that
// are not valid hash sizes (e.g. 50 zeros). Compiled once at package init to
// avoid repeated allocations in hot paths such as webhook delivery.
var zeroHashPattern = regexp.MustCompile(`^(0{40}|0{64})$`)

// IsZeroHash returns whether the hash is a zero hash.
func IsZeroHash(h string) bool {
	return zeroHashPattern.MatchString(h)
}

// Commit is a wrapper around git.Commit with helper methods.
type Commit = git.Commit

// Commits is a list of commits.
type Commits []*Commit

// Len implements sort.Interface.
func (cl Commits) Len() int { return len(cl) }

// Swap implements sort.Interface.
func (cl Commits) Swap(i, j int) { cl[i], cl[j] = cl[j], cl[i] }

// Less implements sort.Interface.
// Sorts by author date (not committer date) so the displayed order matches
// the original authorship timeline. Cherry-picks and rebases will therefore
// appear at the position of their original authoring, not the rebase time.
func (cl Commits) Less(i, j int) bool {
	return cl[i].Author.When.After(cl[j].Author.When)
}
