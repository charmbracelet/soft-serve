package git

import git "github.com/aymanbagabas/git-module"

// StashDiff returns the diff of the given stash index.
func (r *Repository) StashDiff(index int) (*Diff, error) {
	diff, err := r.Repository.StashDiff(index, DiffMaxFiles, DiffMaxFileLines, DiffMaxLineChars, git.DiffOptions{
		CommandOptions: git.CommandOptions{
			Envs: []string{"GIT_CONFIG_GLOBAL=/dev/null"},
		},
	})
	if err != nil {
		return nil, err
	}
	return toDiff(diff), nil
}
