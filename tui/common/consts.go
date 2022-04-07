package common

import (
	"time"

	"github.com/charmbracelet/bubbles/key"
)

// Some constants were copied from https://docs.gitea.io/en-us/config-cheat-sheet/#git-git

const (
	GlamourMaxWidth  = 120
	RepoNameMaxWidth = 32
	MaxDiffLines     = 1000
	MaxDiffFiles     = 100
	MaxPatchWait     = time.Second * 3
)

var (
	PrevPage = key.NewBinding(
		key.WithKeys("pgup", "b", "u"),
		key.WithHelp("pgup", "prev page"),
	)
	NextPage = key.NewBinding(
		key.WithKeys("pgdown", "f", "d"),
		key.WithHelp("pgdn", "next page"),
	)
)
