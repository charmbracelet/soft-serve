// Package repo provides repository UI pages.
package repo

import (
	"fmt"

	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/ui/common"
)

func defaultEmptyRepoMsg(cfg *config.Config, repo string) string {
	return fmt.Sprintf(`# Quick Start

Get started by cloning this repository, add your files, commit, and push.

## Clone this repository.

`+"```"+`sh
git clone %[1]s
`+"```"+`

## Creating a new repository on the command line

`+"```"+`sh
touch README.md
git init
git add README.md
git branch -M main
git commit -m "first commit"
git remote add origin %[1]s
git push -u origin main
`+"```"+`

## Pushing an existing repository from the command line

`+"```"+`sh
git remote add origin %[1]s
git push -u origin main
`+"```"+`
`, common.RepoURL(cfg.SSH.PublicURL, repo))
}
