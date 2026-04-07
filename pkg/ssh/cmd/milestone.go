package cmd

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/spf13/cobra"
)

// milestoneCommand returns a command for managing repository milestones.
func milestoneCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "milestone",
		Aliases: []string{"milestones"},
		Short:   "Manage repository milestones",
	}

	cmd.AddCommand(
		milestoneCreateCommand(),
		milestoneListCommand(),
		milestoneViewCommand(),
		milestoneEditCommand(),
		milestoneCloseCommand(),
		milestoneReopenCommand(),
		milestoneDeleteCommand(),
	)

	return cmd
}

// milestoneCreateCommand creates a new milestone in a repository.
func milestoneCreateCommand() *cobra.Command {
	var description string

	cmd := &cobra.Command{
		Use:               "create REPOSITORY TITLE",
		Short:             "Create a new milestone",
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repoName := args[0]
			title := args[1]

			ms, err := be.CreateMilestone(ctx, repoName, title, description)
			if err != nil {
				return err
			}

			cmd.Printf("Milestone #%d created\n", ms.ID())
			return nil
		},
	}

	cmd.Flags().StringVarP(&description, "description", "d", "", "Milestone description")

	return cmd
}

// milestoneListCommand lists milestones for a repository.
func milestoneListCommand() *cobra.Command {
	var closed bool

	cmd := &cobra.Command{
		Use:               "list REPOSITORY",
		Aliases:           []string{"ls"},
		Short:             "List milestones",
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: checkIfReadable,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repoName := args[0]

			milestones, err := be.ListMilestones(ctx, repoName, !closed)
			if err != nil {
				return err
			}

			if len(milestones) == 0 {
				cmd.Println("No milestones found")
				return nil
			}

			cmd.Printf("%-6s %-8s %-20s %s\n", "#", "STATUS", "CREATED", "TITLE")
			for _, ms := range milestones {
				status := "open"
				if ms.IsClosed() {
					status = "closed"
				}
				created := ms.CreatedAt().Format("2006-01-02")
				title := ms.Title()
				if len(title) > 40 {
					title = title[:37] + "..."
				}
				cmd.Printf("%-6d %-8s %-20s %s\n", ms.ID(), status, created, title)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&closed, "closed", false, "Show closed milestones instead of open")

	return cmd
}

// milestoneViewCommand views a milestone.
func milestoneViewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "view REPOSITORY MILESTONE_ID",
		Short:             "View a milestone",
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: checkIfReadable,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repoName := args[0]
			milestoneID, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid milestone ID: %s", args[1])
			}

			ms, err := be.GetMilestone(ctx, repoName, milestoneID)
			if err != nil {
				return err
			}

			status := "open"
			if ms.IsClosed() {
				status = "closed"
			}

			cmd.Printf("Milestone #%d: %s\n", ms.ID(), ms.Title())
			cmd.Printf("Status: %s\n", status)
			if ms.Description() != "" {
				cmd.Printf("Description: %s\n", ms.Description())
			}
			if !ms.DueDate().IsZero() {
				cmd.Printf("Due: %s\n", ms.DueDate().Format("2006-01-02"))
			}
			cmd.Printf("Created: %s\n", ms.CreatedAt().Format("2006-01-02 15:04:05"))
			return nil
		},
	}

	return cmd
}

// milestoneEditCommand edits an existing milestone.
func milestoneEditCommand() *cobra.Command {
	var title string
	var description string

	cmd := &cobra.Command{
		Use:               "edit REPOSITORY MILESTONE_ID",
		Short:             "Edit a milestone",
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repoName := args[0]
			milestoneID, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid milestone ID: %s", args[1])
			}

			ms, err := be.GetMilestone(ctx, repoName, milestoneID)
			if err != nil {
				return err
			}

			newTitle := ms.Title()
			if cmd.Flags().Changed("title") {
				newTitle = title
			}
			newDesc := ms.Description()
			if cmd.Flags().Changed("description") {
				newDesc = description
			}

			if err := be.UpdateMilestone(ctx, repoName, milestoneID, newTitle, newDesc); err != nil {
				return err
			}

			cmd.Printf("Milestone #%d updated\n", milestoneID)
			return nil
		},
	}

	cmd.Flags().StringVarP(&title, "title", "t", "", "New milestone title")
	cmd.Flags().StringVarP(&description, "description", "d", "", "New milestone description")

	return cmd
}

// milestoneCloseCommand closes a milestone.
func milestoneCloseCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "close REPOSITORY MILESTONE_ID",
		Short:             "Close a milestone",
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repoName := args[0]
			milestoneID, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid milestone ID: %s", args[1])
			}

			if err := be.CloseMilestone(ctx, repoName, milestoneID); err != nil {
				return err
			}

			cmd.Printf("Milestone #%d closed\n", milestoneID)
			return nil
		},
	}

	return cmd
}

// milestoneReopenCommand reopens a closed milestone.
func milestoneReopenCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "reopen REPOSITORY MILESTONE_ID",
		Short:             "Reopen a milestone",
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repoName := args[0]
			milestoneID, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid milestone ID: %s", args[1])
			}

			if err := be.ReopenMilestone(ctx, repoName, milestoneID); err != nil {
				return err
			}

			cmd.Printf("Milestone #%d reopened\n", milestoneID)
			return nil
		},
	}

	return cmd
}

// milestoneDeleteCommand deletes a milestone from a repository.
func milestoneDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "delete REPOSITORY MILESTONE_ID",
		Short:             "Delete a milestone",
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repoName := args[0]
			milestoneID, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid milestone ID: %s", args[1])
			}

			if err := be.DeleteMilestone(ctx, repoName, milestoneID); err != nil {
				return err
			}

			cmd.Printf("Milestone #%d deleted\n", milestoneID)
			return nil
		},
	}

	return cmd
}

// issueMilestoneCommand returns a command for managing the milestone on an issue.
func issueMilestoneCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "milestone",
		Short: "Manage milestone on an issue",
	}

	cmd.AddCommand(
		issueMilestoneSetCommand(),
		issueMilestoneUnsetCommand(),
	)

	return cmd
}

// issueMilestoneSetCommand sets the milestone for an issue.
func issueMilestoneSetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "set REPOSITORY ISSUE_ID MILESTONE_ID",
		Short:             "Set milestone on an issue",
		Args:              cobra.ExactArgs(3),
		PersistentPreRunE: checkIfCollab,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repoName := args[0]
			issueID, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid issue ID: %s", args[1])
			}
			milestoneID, err := strconv.ParseInt(args[2], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid milestone ID: %s", args[2])
			}

			if _, _, err := getIssueAndVerifyRepo(cmd, be, repoName, issueID); err != nil {
				return err
			}

			// Verify milestone exists in this repo.
			if _, err := be.GetMilestone(ctx, repoName, milestoneID); err != nil {
				return err
			}

			user := proto.UserFromContext(ctx)
			if user == nil {
				return fmt.Errorf("user not found")
			}
			if be.AccessLevelForUser(ctx, repoName, user) < access.ReadWriteAccess {
				return fmt.Errorf("unauthorized: only collaborators and admins can set milestones")
			}

			if err := be.SetIssueMilestone(ctx, issueID, milestoneID); err != nil {
				return err
			}

			cmd.Printf("Milestone #%d set on issue #%d\n", milestoneID, issueID)
			return nil
		},
	}

	return cmd
}

// issueMilestoneUnsetCommand removes the milestone from an issue.
func issueMilestoneUnsetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "unset REPOSITORY ISSUE_ID",
		Short:             "Unset milestone from an issue",
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: checkIfCollab,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repoName := args[0]
			issueID, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid issue ID: %s", args[1])
			}

			if _, _, err := getIssueAndVerifyRepo(cmd, be, repoName, issueID); err != nil {
				return err
			}

			user := proto.UserFromContext(ctx)
			if user == nil {
				return fmt.Errorf("user not found")
			}
			if be.AccessLevelForUser(ctx, repoName, user) < access.ReadWriteAccess {
				return fmt.Errorf("unauthorized: only collaborators and admins can unset milestones")
			}

			if err := be.UnsetIssueMilestone(ctx, issueID); err != nil {
				return err
			}

			cmd.Printf("Milestone unset from issue #%d\n", issueID)
			return nil
		},
	}

	return cmd
}
