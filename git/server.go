package git

import (
	"context"

	"github.com/aymanbagabas/git-module"
)

// UpdateServerInfo updates the server info file for the given repo path.
func UpdateServerInfo(ctx context.Context, path string) error {
	if !isGitDir(path) {
		return ErrNotAGitRepository
	}

	cmd := git.NewCommand("update-server-info").WithContext(ctx).WithTimeout(-1)
	_, err := cmd.RunInDir(path)
	return err
}
