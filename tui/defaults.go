package tui

import (
	"os"
	"path/filepath"
	"smoothie/git"

	gg "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

const defaultReadme = "# Smoothie\n\n Welcome! You can configure your Smoothie server by cloning this repo and pushing changes.\n\n## Repos\n\n{{ range .Menu }}* {{ .Name }}{{ if .Note }} - {{ .Note }} {{ end }}\n  - `git clone ssh://{{$.Host}}:{{$.Port}}/{{.Repo}}`\n{{ end }}"

const defaultConfig = `{
	"name": "Smoothie",
	"show_all_repos": true,
	"host": "localhost",
	"port": 23231,
	"menu": [
	  {
			"name": "Home",
			"repo": "config",
			"note": "Configuration and content repo for this server"
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
