package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/soft-serve/git"
	gitm "github.com/gogs/git-module"
	"github.com/spf13/cobra"
)

func branchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "branch",
		Short: "Manage repository branches",
	}

	cmd.AddCommand(
		branchListCommand(),
		branchDefaultCommand(),
		branchDeleteCommand(),
	)

	return cmd
}

func branchListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "list REPOSITORY",
		Short:             "List repository branches",
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: checkIfReadable,
		RunE: func(cmd *cobra.Command, args []string) error {
			be, _ := fromContext(cmd)
			rn := strings.TrimSuffix(args[0], ".git")
			ctx := cmd.Context()
			rr, err := be.Repository(ctx, rn)
			if err != nil {
				return err
			}

			r, err := rr.Open()
			if err != nil {
				return err
			}

			branches, _ := r.Branches()
			for _, b := range branches {
				cmd.Println(b)
			}

			return nil
		},
	}

	return cmd
}

func branchDefaultCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "default REPOSITORY [BRANCH]",
		Short: "Set or get the default branch",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be, _ := fromContext(cmd)
			rn := strings.TrimSuffix(args[0], ".git")
			switch len(args) {
			case 1:
				if err := checkIfReadable(cmd, args); err != nil {
					return err
				}

				rr, err := be.Repository(ctx, rn)
				if err != nil {
					return err
				}

				r, err := rr.Open()
				if err != nil {
					return err
				}

				head, err := r.HEAD()
				if err != nil {
					return err
				}

				cmd.Println(head.Name().Short())
			case 2:
				if err := checkIfCollab(cmd, args); err != nil {
					return err
				}

				rr, err := be.Repository(ctx, rn)
				if err != nil {
					return err
				}

				r, err := rr.Open()
				if err != nil {
					return err
				}

				branch := args[1]
				branches, _ := r.Branches()
				var exists bool
				for _, b := range branches {
					if branch == b {
						exists = true
						break
					}
				}

				if !exists {
					return git.ErrReferenceNotExist
				}

				if _, err := r.SymbolicRef("HEAD", gitm.RefsHeads+branch); err != nil {
					return err
				}
			}

			return nil
		},
	}

	return cmd
}

func branchDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "delete REPOSITORY BRANCH",
		Aliases:           []string{"remove", "rm", "del"},
		Short:             "Delete a branch",
		PersistentPreRunE: checkIfCollab,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be, _ := fromContext(cmd)
			rn := strings.TrimSuffix(args[0], ".git")
			rr, err := be.Repository(ctx, rn)
			if err != nil {
				return err
			}

			r, err := rr.Open()
			if err != nil {
				return err
			}

			branch := args[1]
			branches, _ := r.Branches()
			var exists bool
			for _, b := range branches {
				if branch == b {
					exists = true
					break
				}
			}

			if !exists {
				return git.ErrReferenceNotExist
			}

			head, err := r.HEAD()
			if err != nil {
				return err
			}

			if head.Name().Short() == branch {
				return fmt.Errorf("cannot delete the default branch")
			}

			if err := r.DeleteBranch(branch, gitm.DeleteBranchOptions{Force: true}); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}
