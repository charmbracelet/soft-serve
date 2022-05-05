package git

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/soft-serve/git"
)

// ErrMissingRepo indicates that the requested repository could not be found.
var ErrMissingRepo = errors.New("missing repo")

// GitRepo is an interface for Git repositories.
type GitRepo interface {
	Repo() string
	Name() string
	Description() string
	Readme() (string, string)
	HEAD() (*git.Reference, error)
	CommitsByPage(*git.Reference, int, int) (git.Commits, error)
	CountCommits(*git.Reference) (int64, error)
	Diff(*git.Commit) (*git.Diff, error)
	References() ([]*git.Reference, error)
	Tree(*git.Reference, string) (*git.Tree, error)
	IsPrivate() bool
}

// GitRepoSource is an interface for Git repository factory.
type GitRepoSource interface {
	GetRepo(string) (GitRepo, error)
	AllRepos() []GitRepo
}

// RepoURL returns the URL of the repository.
func RepoURL(host string, port int, name string) string {
	p := ""
	if port != 22 {
		p += fmt.Sprintf(":%d", port)
	}
	return fmt.Sprintf("git clone ssh://%s/%s", host+p, name)
}
