package config

import (
	"errors"
	"path/filepath"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/storage/memory"
)

const (
	defaultConfigRepo = "config"
	defaultReadme     = "# Soft Serve\n\n Welcome! You can configure your Soft Serve server by cloning this repo and pushing changes.\n\n```\ngit clone ssh://localhost:23231/config\n```"
)

func (cfg *Config) createDefaultConfigRepoAndUsers() error {
	rp := filepath.Join(cfg.RepoPath(), defaultConfigRepo) + ".git"
	_, err := gogit.PlainOpen(rp)
	if errors.Is(err, gogit.ErrRepositoryNotExists) {
		if err := cfg.Create(defaultConfigRepo, "Config", "Soft Serve Config", true); err != nil {
			return err
		}
		repo, err := gogit.Clone(memory.NewStorage(), memfs.New(), &gogit.CloneOptions{
			URL: rp,
		})
		if err != nil && err != transport.ErrEmptyRemoteRepository {
			return err
		}
		wt, err := repo.Worktree()
		if err != nil {
			return err
		}
		rm, err := wt.Filesystem.Create("README.md")
		if err != nil {
			return err
		}
		_, err = rm.Write([]byte(defaultReadme))
		if err != nil {
			return err
		}
		_, err = wt.Add("README.md")
		if err != nil {
			return err
		}
		author := object.Signature{
			Name:  "Soft Serve Server",
			Email: "vt100@charm.sh",
			When:  time.Now(),
		}
		_, err = wt.Commit("Default init", &gogit.CommitOptions{
			All:       true,
			Author:    &author,
			Committer: &author,
		})
		if err != nil {
			return err
		}
		err = repo.Push(&gogit.PushOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}
