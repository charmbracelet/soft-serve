package cmd

import (
	"io"
	"os/exec"

	"github.com/charmbracelet/soft-serve/proto"
	"github.com/spf13/cobra"
)

// TODO: remove this command.
// GitCommand returns a command that handles Git operations.
func GitCommand() *cobra.Command {
	gitCmd := &cobra.Command{
		Use:   "git REPO COMMAND",
		Short: "Perform Git operations on a repository.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, s := fromContext(cmd)
			auth := cfg.AuthRepo("config", s.PublicKey())
			if auth < proto.AdminAccess {
				return ErrUnauthorized
			}
			if len(args) < 1 {
				return runGit(nil, s, s, "")
			}
			var repo proto.Repository
			rn := args[0]
			repoExists := false
			repos, err := cfg.ListRepos()
			if err != nil {
				return err
			}
			for _, rp := range repos {
				if rp.Name() == rn {
					re, err := rp.Open()
					if err != nil {
						continue
					}
					repoExists = true
					repo = re
					break
				}
			}
			if !repoExists {
				return ErrRepoNotFound
			}
			return runGit(nil, s, s, repo.Repository().Path, args[1:]...)
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
