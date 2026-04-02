package cmd

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/spf13/cobra"
)

// issueAssigneeCommand returns a command for managing issue assignees.
func issueAssigneeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "assignee",
		Short: "Manage issue assignees",
	}

	cmd.AddCommand(
		issueAssigneeAddCommand(),
		issueAssigneeRemoveCommand(),
	)

	return cmd
}

// issueAssigneeAddCommand returns a command for adding an assignee to an issue.
func issueAssigneeAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "add REPOSITORY ISSUE_ID USERNAME",
		Short:             "Assign a user to an issue",
		Args:              cobra.ExactArgs(3),
		PersistentPreRunE: checkIfReadable,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repoName := args[0]
			issueID, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid issue ID: %s", args[1])
			}
			username := args[2]

			user := proto.UserFromContext(ctx)
			if user == nil {
				return fmt.Errorf("user not found")
			}
			if be.AccessLevelForUser(ctx, repoName, user) < access.ReadWriteAccess {
				return fmt.Errorf("unauthorized: you don't have write access to %s", repoName)
			}

			if err := be.AssignUserToIssue(ctx, repoName, issueID, username); err != nil {
				return err
			}

			cmd.Printf("User %q assigned to issue #%d\n", username, issueID)
			return nil
		},
	}

	return cmd
}

// issueAssigneeRemoveCommand returns a command for removing an assignee from an issue.
func issueAssigneeRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "remove REPOSITORY ISSUE_ID USERNAME",
		Short:             "Unassign a user from an issue",
		Args:              cobra.ExactArgs(3),
		PersistentPreRunE: checkIfReadable,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repoName := args[0]
			issueID, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid issue ID: %s", args[1])
			}
			username := args[2]

			user := proto.UserFromContext(ctx)
			if user == nil {
				return fmt.Errorf("user not found")
			}
			if be.AccessLevelForUser(ctx, repoName, user) < access.ReadWriteAccess {
				return fmt.Errorf("unauthorized: you don't have write access to %s", repoName)
			}

			if err := be.UnassignUserFromIssue(ctx, repoName, issueID, username); err != nil {
				return err
			}

			cmd.Printf("User %q unassigned from issue #%d\n", username, issueID)
			return nil
		},
	}

	return cmd
}
