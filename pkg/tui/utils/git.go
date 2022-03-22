package utils

import (
	"github.com/charmbracelet/soft-serve/pkg/git"
)

type GitRepo interface {
	Name() string
	Readme() (string, string)
	HEAD() (*git.Reference, error)
	CommitsByPage(*git.Reference, int, int) (git.Commits, error)
	CountCommits(*git.Reference) (int64, error)
	Diff(*git.Commit) (*git.Diff, error)
	References() ([]*git.Reference, error)
	Tree(*git.Reference, string) (*git.Tree, error)
}
