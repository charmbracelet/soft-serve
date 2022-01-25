package types

import (
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type Repo interface {
	Name() string
	GetReference() *plumbing.Reference
	SetReference(*plumbing.Reference) error
	GetReadme() string
	GetCommits(limit int) Commits
	Repository() *git.Repository
	Tree(path string) (*object.Tree, error)
}

type Commit struct {
	*object.Commit
}

type Commits []*Commit

func (cl Commits) Len() int      { return len(cl) }
func (cl Commits) Swap(i, j int) { cl[i], cl[j] = cl[j], cl[i] }
func (cl Commits) Less(i, j int) bool {
	return cl[i].Author.When.After(cl[j].Author.When)
}
