package ui

import (
	"github.com/charmbracelet/soft-serve/config"
	"github.com/charmbracelet/soft-serve/ui/git"
)

// source is a wrapper around config.RepoSource that implements git.GitRepoSource.
type source struct {
	*config.RepoSource
}

// GetRepo implements git.GitRepoSource.
func (s *source) GetRepo(name string) (git.GitRepo, error) {
	return s.RepoSource.GetRepo(name)
}

// AllRepos implements git.GitRepoSource.
func (s *source) AllRepos() []git.GitRepo {
	rs := make([]git.GitRepo, 0)
	for _, r := range s.RepoSource.AllRepos() {
		rs = append(rs, r)
	}
	return rs
}
