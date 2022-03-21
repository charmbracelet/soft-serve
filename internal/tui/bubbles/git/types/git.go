package types

import (
	"github.com/gogs/git-module"
)

type Repo interface {
	Name() string
	GetHEAD() *git.Reference
	GetReferences() []*git.Reference
	GetReadme() string
	GetReadmePath() string
	Count(*git.Reference) (int64, error)
	GetCommitsByPage(*git.Reference, int, int) (Commits, error)
	Tree(*git.Reference, string) (*git.Tree, error)
	TreeEntryFile(*git.TreeEntry) (*git.Blob, error)
	Patch(hash string) (*git.Diff, error)
}

type Commits []*git.Commit

func (cl Commits) Len() int      { return len(cl) }
func (cl Commits) Swap(i, j int) { cl[i], cl[j] = cl[j], cl[i] }
func (cl Commits) Less(i, j int) bool {
	return cl[i].Author.When.After(cl[j].Author.When)
}
