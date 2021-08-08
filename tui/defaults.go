package tui

import (
	"os"
	"path/filepath"
	"smoothie/git"

	gg "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

const defaultReadme = "# Smoothie\nWelcome to Smoothie. To setup your own configuration, please clone this repo."

const defaultConfig = `{
	"name": "Smoothie",
	"show_all_repos": true,
	"menu": [
	  {
			"name": "Home",
			"repo": "config"
		}
	]
}`

func createFile(path string, content string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	if err != nil {
		return err
	}
	return f.Sync()
}

func createDefaultConfigRepo(rs *git.RepoSource) error {
	cn := "config"
	err := rs.LoadRepos()
	cr, err := rs.GetRepo(cn)
	if err == git.ErrMissingRepo {
		cr, err = rs.InitRepo(cn, false)
		if err != nil {
			return err
		}

		rp := filepath.Join(rs.Path, cn, "README.md")
		err = createFile(rp, defaultReadme)
		if err != nil {
			return err
		}
		cp := filepath.Join(rs.Path, cn, "config.json")
		err = createFile(cp, defaultConfig)
		if err != nil {
			return err
		}
		wt, err := cr.Repository.Worktree()
		if err != nil {
			return err
		}
		_, err = wt.Add("README.md")
		if err != nil {
			return err
		}
		_, err = wt.Add("config.json")
		if err != nil {
			return err
		}
		_, err = wt.Commit("Default init", &gg.CommitOptions{
			All: true,
			Author: &object.Signature{
				Name:  "Smoothie Server",
				Email: "vt100@charm.sh",
			},
		})
		if err != nil {
			return err
		}
		err = rs.LoadRepos()
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}
