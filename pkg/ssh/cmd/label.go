package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/spf13/cobra"
)

// labelCommand returns a command for managing repository labels.
func labelCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "label",
		Aliases: []string{"labels"},
		Short:   "Manage repository labels",
	}

	cmd.AddCommand(
		labelCreateCommand(),
		labelListCommand(),
		labelEditCommand(),
		labelDeleteCommand(),
	)

	return cmd
}

// labelCreateCommand creates a new label in a repository.
func labelCreateCommand() *cobra.Command {
	var color string
	var description string

	cmd := &cobra.Command{
		Use:               "create REPOSITORY NAME",
		Short:             "Create a new label",
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repoName := args[0]
			name := args[1]

			lbl, err := be.CreateLabel(ctx, repoName, name, color, description)
			if err != nil {
				return err
			}

			cmd.Printf("Label %q created (ID %d)\n", lbl.Name(), lbl.ID())
			return nil
		},
	}

	cmd.Flags().StringVarP(&color, "color", "c", "", "Label color as a hex string (e.g. #ff0000)")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Label description")

	return cmd
}

// labelListCommand lists labels for a repository.
func labelListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "list REPOSITORY",
		Aliases:           []string{"ls"},
		Short:             "List labels",
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: checkIfReadable,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repoName := args[0]

			labels, err := be.ListLabels(ctx, repoName)
			if err != nil {
				return err
			}

			if len(labels) == 0 {
				cmd.Println("No labels found")
				return nil
			}

			cmd.Printf("%-4s %-20s %-10s %s\n", "ID", "NAME", "COLOR", "DESCRIPTION")
			for _, l := range labels {
				color := l.Color()
				if color == "" {
					color = "-"
				}
				desc := l.Description()
				if len(desc) > 40 {
					desc = desc[:37] + "..."
				}
				cmd.Printf("%-4d %-20s %-10s %s\n", l.ID(), l.Name(), color, desc)
			}
			return nil
		},
	}

	return cmd
}

// labelEditCommand edits an existing label.
func labelEditCommand() *cobra.Command {
	var name string
	var color string
	var description string

	cmd := &cobra.Command{
		Use:               "edit REPOSITORY LABEL_ID",
		Short:             "Edit a label",
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repoName := args[0]
			labelID, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid label ID: %s", args[1])
			}

			lbl, err := getLabelAndVerifyRepo(be, cmd, repoName, labelID)
			if err != nil {
				return err
			}

			newName := lbl.Name()
			if cmd.Flags().Changed("name") {
				newName = strings.TrimSpace(name)
			}
			newColor := lbl.Color()
			if cmd.Flags().Changed("color") {
				newColor = color
			}
			newDesc := lbl.Description()
			if cmd.Flags().Changed("description") {
				newDesc = description
			}

			if err := be.UpdateLabel(ctx, repoName, labelID, newName, newColor, newDesc); err != nil {
				return err
			}

			cmd.Printf("Label %d updated\n", labelID)
			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "New label name")
	cmd.Flags().StringVarP(&color, "color", "c", "", "New label color")
	cmd.Flags().StringVarP(&description, "description", "d", "", "New label description")

	return cmd
}

// labelDeleteCommand deletes a label from a repository.
func labelDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "delete REPOSITORY LABEL_ID",
		Short:             "Delete a label",
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repoName := args[0]
			labelID, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid label ID: %s", args[1])
			}

			if _, err := getLabelAndVerifyRepo(be, cmd, repoName, labelID); err != nil {
				return err
			}

			if err := be.DeleteLabel(ctx, repoName, labelID); err != nil {
				return err
			}

			cmd.Printf("Label %d deleted\n", labelID)
			return nil
		},
	}

	return cmd
}

// getLabelAndVerifyRepo fetches a label and verifies it belongs to the given repository.
func getLabelAndVerifyRepo(be *backend.Backend, cmd *cobra.Command, repoName string, labelID int64) (proto.Label, error) {
	ctx := cmd.Context()

	repo, err := be.Repository(ctx, repoName)
	if err != nil {
		return nil, err
	}

	labels, err := be.ListLabels(ctx, repoName)
	if err != nil {
		return nil, err
	}

	for _, l := range labels {
		if l.ID() == labelID && l.RepoID() == repo.ID() {
			return l, nil
		}
	}

	return nil, fmt.Errorf("label %d not found in repository %s", labelID, repoName)
}

// issueLabelCommand returns a command for managing labels on issues.
func issueLabelCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "label",
		Short: "Manage labels on an issue",
	}

	cmd.AddCommand(
		issueLabelAddCommand(),
		issueLabelRemoveCommand(),
	)

	return cmd
}

// issueLabelAddCommand attaches a label to an issue.
func issueLabelAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "add REPOSITORY ISSUE_ID LABEL_NAME",
		Short:             "Add a label to an issue",
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
			labelName := args[2]

			issue, _, err := getIssueAndVerifyRepo(cmd, be, repoName, issueID)
			if err != nil {
				return err
			}

			// Verify the caller is at least a collaborator.
			user := proto.UserFromContext(ctx)
			if user == nil {
				return fmt.Errorf("user not found")
			}
			if be.AccessLevelForUser(ctx, repoName, user) < access.ReadWriteAccess {
				return fmt.Errorf("unauthorized: only collaborators and admins can add labels")
			}

			lbl, err := be.GetLabel(ctx, repoName, labelName)
			if err != nil {
				return err
			}

			if err := be.AddLabelToIssue(ctx, issue.ID(), lbl.ID()); err != nil {
				return err
			}

			cmd.Printf("Label %q added to issue #%d\n", lbl.Name(), issue.ID())
			return nil
		},
	}

	return cmd
}

// issueLabelRemoveCommand detaches a label from an issue.
func issueLabelRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "remove REPOSITORY ISSUE_ID LABEL_NAME",
		Short:             "Remove a label from an issue",
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
			labelName := args[2]

			issue, _, err := getIssueAndVerifyRepo(cmd, be, repoName, issueID)
			if err != nil {
				return err
			}

			user := proto.UserFromContext(ctx)
			if user == nil {
				return fmt.Errorf("user not found")
			}
			if be.AccessLevelForUser(ctx, repoName, user) < access.ReadWriteAccess {
				return fmt.Errorf("unauthorized: only collaborators and admins can remove labels")
			}

			lbl, err := be.GetLabel(ctx, repoName, labelName)
			if err != nil {
				return err
			}

			if err := be.RemoveLabelFromIssue(ctx, issue.ID(), lbl.ID()); err != nil {
				return err
			}

			cmd.Printf("Label %q removed from issue #%d\n", lbl.Name(), issue.ID())
			return nil
		},
	}

	return cmd
}
