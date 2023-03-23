package repo

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/soft-serve/server/config"
)

func defaultEmptyRepoMsg(cfg *config.Config, repo string) string {
	host := cfg.Backend.ServerHost()
	if cfg.Backend.ServerPort() != "22" {
		host = fmt.Sprintf("%s:%s", host, cfg.Backend.ServerPort())
	}
	repo = strings.TrimSuffix(repo, ".git")
	return fmt.Sprintf(`# Quick Start

Get started by cloning this repository, add your files, commit, and push.

## Clone this repository.

`+"```"+`sh
git clone ssh://%[1]s/%[2]s.git
`+"```"+`

## Creating a new repository on the command line

`+"```"+`sh
touch README.md
git init
git add README.md
git branch -M main
git commit -m "first commit"
git remote add origin ssh://%[1]s/%[2]s.git
git push -u origin main
`+"```"+`

## Pushing an existing repository from the command line

`+"```"+`sh
git remote add origin ssh://%[1]s/%[2]s.git
git push -u origin main
`+"```"+`
`, host, repo)
}
