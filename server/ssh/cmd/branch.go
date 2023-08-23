package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/proto"
	"github.com/charmbracelet/soft-serve/server/webhook"
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
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			rn := strings.TrimSuffix(args[0], ".git")
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
			be := backend.FromContext(ctx)
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

				if _, err := r.SymbolicRef(git.HEAD, gitm.RefsHeads+branch, gitm.SymbolicRefOptions{
					CommandOptions: gitm.CommandOptions{
						Context: ctx,
					},
				}); err != nil {
					return err
				}

				// TODO: move this to backend?
				user := proto.UserFromContext(ctx)
				wh, err := webhook.NewRepositoryEvent(ctx, user, rr, webhook.RepositoryEventActionDefaultBranchChange)
				if err != nil {
					return err
				}

				return webhook.SendEvent(ctx, wh)
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
			be := backend.FromContext(ctx)
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

			branchCommit, err := r.BranchCommit(branch)
			if err != nil {
				return err
			}

			if err := r.DeleteBranch(branch, gitm.DeleteBranchOptions{Force: true}); err != nil {
				return err
			}

			wh, err := webhook.NewBranchTagEvent(ctx, proto.UserFromContext(ctx), rr, git.RefsHeads+branch, branchCommit.ID.String(), git.ZeroID)
			if err != nil {
				return err
			}

			return webhook.SendEvent(ctx, wh)
		},
	}

	return cmd
}
