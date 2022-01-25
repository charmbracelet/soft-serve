package tui

import (
	"path/filepath"

	gitypes "github.com/charmbracelet/soft-serve/internal/tui/bubbles/git/types"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type Repo struct {
	name   string
	repo   *git.Repository
	readme string
	ref    *plumbing.Reference
}

func (r *Repo) Name() string {
	return r.name
}

func (r *Repo) GetReference() *plumbing.Reference {
	return r.ref
}

func (r *Repo) SetReference(ref *plumbing.Reference) error {
	r.ref = ref
	return nil
}

func (r *Repo) Repository() *git.Repository {
	return r.repo
}

func (r *Repo) Tree(path string) (*object.Tree, error) {
	path = filepath.Clean(path)
	c, err := r.repo.CommitObject(r.ref.Hash())
	if err != nil {
		return nil, err
	}
	t, err := c.Tree()
	if err != nil {
		return nil, err
	}
	if path == "." {
		return t, nil
	}
	return t.Tree(path)
}

func (r *Repo) GetCommits(limit int) (gitypes.Commits, error) {
	commits := gitypes.Commits{}
	l, err := r.repo.Log(&git.LogOptions{
		Order: git.LogOrderCommitterTime,
		From:  r.ref.Hash(),
	})
	if err != nil {
		return nil, err
	}
	err = l.ForEach(func(c *object.Commit) error {
		commits = append(commits, &gitypes.Commit{c})
		return nil
	})
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > len(commits) {
		limit = len(commits)
	}
	return commits[:limit], nil
}

func (r *Repo) GetReadme() string {
	if r.readme != "" {
		return r.readme
	}
	md, err := r.readFile("README.md")
	if err != nil {
		return ""
	}
	return md
}

func (r *Repo) readFile(path string) (string, error) {
	lg, err := r.repo.Log(&git.LogOptions{
		From: r.ref.Hash(),
	})
	if err != nil {
		return "", err
	}
	c, err := lg.Next()
	if err != nil {
		return "", err
	}
	f, err := c.File(path)
	if err != nil {
		return "", err
	}
	content, err := f.Contents()
	if err != nil {
		return "", err
	}
	return content, nil
}
