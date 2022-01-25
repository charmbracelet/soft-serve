package types

import "time"

// Some constants were copied from https://docs.gitea.io/en-us/config-cheat-sheet/#git-git

const (
	GlamourMaxWidth  = 120
	RepoNameMaxWidth = 32
	MaxDiffLines     = 1000
	MaxDiffFiles     = 100
	MaxPatchWait     = time.Second * 3
)
