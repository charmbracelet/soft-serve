package cmd

import (
	"io"
	"os/exec"

	"github.com/charmbracelet/soft-serve/config"
	gitwish "github.com/charmbracelet/wish/git"
	"github.com/spf13/cobra"
)

// GitCommand returns a command that handles Git operations.
func GitCommand() *cobra.Command {
	gitCmd := &cobra.Command{
		Use:   "git REPO COMMAND",
		Short: "Perform Git operations on a repository.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, s := fromContext(cmd)
			auth := ac.AuthRepo("config", s.PublicKey())
			if auth < gitwish.AdminAccess {
				return ErrUnauthorized
			}
			if len(args) < 1 {
				return runGit(nil, s, s, "")
			}
			var repo *config.Repo
			rn := args[0]
			repoExists := false
			for _, rp := range ac.Source.AllRepos() {
				if rp.Repo() == rn {
					repoExists = true
					repo = rp
					break
				}
			}
			if !repoExists {
				return ErrRepoNotFound
			}
			return runGit(nil, s, s, repo.Path(), args[1:]...)
		},
	}
	gitCmd.Flags().SetInterspersed(false)

	return gitCmd
}

func runGit(in io.Reader, out, err io.Writer, dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = err
	cmd.Dir = dir
	return cmd.Run()
}
