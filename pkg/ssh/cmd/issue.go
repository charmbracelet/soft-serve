package cmd

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/spf13/cobra"
)

// issueCommand returns a command for managing issues.
func issueCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "issue",
		Aliases: []string{"issues"},
		Short:   "Manage issues",
	}

	cmd.AddCommand(
		issueCreateCommand(),
		issueListCommand(),
		issueViewCommand(),
		issueCloseCommand(),
		issueReopenCommand(),
	)

	return cmd
}

// issueCreateCommand returns a command for creating an issue.
func issueCreateCommand() *cobra.Command {
	var title string
	var body string

	cmd := &cobra.Command{
		Use:               "create REPOSITORY",
		Short:             "Create a new issue",
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: checkIfCollab,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repoName := args[0]

			if title == "" {
				return fmt.Errorf("title is required")
			}

			user := proto.UserFromContext(ctx)
			if user == nil {
				return fmt.Errorf("user not found")
			}

			issue, err := be.CreateIssue(ctx, repoName, user.ID(), title, body)
			if err != nil {
				return err
			}

			cmd.Printf("Issue #%d created\n", issue.ID())
			return nil
		},
	}

	cmd.Flags().StringVarP(&title, "title", "t", "", "Issue title (required)")
	cmd.Flags().StringVarP(&body, "body", "b", "", "Issue body")
	_ = cmd.MarkFlagRequired("title")

	return cmd
}

// issueListCommand returns a command for listing issues.
func issueListCommand() *cobra.Command {
	var status string

	cmd := &cobra.Command{
		Use:               "list REPOSITORY",
		Aliases:           []string{"ls"},
		Short:             "List issues",
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: checkIfReadable,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repoName := args[0]

			issues, err := be.GetIssuesByRepository(ctx, repoName, status)
			if err != nil {
				return err
			}

			if len(issues) == 0 {
				cmd.Println("No issues found")
				return nil
			}

			cmd.Printf("%-6s %-8s %-20s %s\n", "#", "STATUS", "CREATED", "TITLE")
			for _, issue := range issues {
				created := issue.CreatedAt().Format("2006-01-02")
				title := issue.Title()
				if len(title) > 40 {
					title = title[:37] + "..."
				}
				cmd.Printf("%-6d %-8s %-20s %s\n", issue.ID(), issue.Status(), created, title)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&status, "status", "s", "", "Filter by status (open, closed, or all)")

	return cmd
}

// issueViewCommand returns a command for viewing an issue.
func issueViewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "view REPOSITORY ISSUE_ID",
		Short:             "View an issue",
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: checkIfReadable,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repoName := args[0]
			issueID, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid issue ID: %s", args[1])
			}

			issue, err := be.GetIssue(ctx, issueID)
			if err != nil {
				return err
			}

			// Verify the issue belongs to the specified repository
			repo, err := be.Repository(ctx, repoName)
			if err != nil {
				return err
			}
			if issue.RepoID() != repo.ID() {
				return fmt.Errorf("issue #%d not found in repository %s", issueID, repoName)
			}

			author, _ := be.UserByID(ctx, issue.UserID())
			authorName := "unknown"
			if author != nil {
				authorName = author.Username()
			}

			cmd.Printf("Issue #%d: %s\n", issue.ID(), issue.Title())
			cmd.Printf("Status: %s\n", issue.Status())
			cmd.Printf("Author: %s\n", authorName)
			cmd.Printf("Created: %s\n", issue.CreatedAt().Format("2006-01-02 15:04:05"))
			if issue.IsClosed() {
				closer, _ := be.UserByID(ctx, issue.ClosedBy())
				closerName := "unknown"
				if closer != nil {
					closerName = closer.Username()
				}
				cmd.Printf("Closed by: %s on %s\n", closerName, issue.ClosedAt().Format("2006-01-02 15:04:05"))
			}
			cmd.Println()
			if issue.Body() != "" {
				cmd.Println(issue.Body())
			} else {
				cmd.Println("(no description)")
			}

			return nil
		},
	}

	return cmd
}

// issueCloseCommand returns a command for closing an issue.
func issueCloseCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "close REPOSITORY ISSUE_ID",
		Short:             "Close an issue",
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

			issue, err := be.GetIssue(ctx, issueID)
			if err != nil {
				return err
			}

			// Verify the issue belongs to the specified repository
			repo, err := be.Repository(ctx, repoName)
			if err != nil {
				return err
			}
			if issue.RepoID() != repo.ID() {
				return fmt.Errorf("issue #%d not found in repository %s", issueID, repoName)
			}

			if issue.IsClosed() {
				return fmt.Errorf("issue #%d is already closed", issueID)
			}

			user := proto.UserFromContext(ctx)
			if user == nil {
				return fmt.Errorf("user not found")
			}

			if err := be.CloseIssue(ctx, issueID, user.ID()); err != nil {
				return err
			}

			cmd.Printf("Issue #%d closed\n", issueID)
			return nil
		},
	}

	return cmd
}

// issueReopenCommand returns a command for reopening an issue.
func issueReopenCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "reopen REPOSITORY ISSUE_ID",
		Short:             "Reopen a closed issue",
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

			issue, err := be.GetIssue(ctx, issueID)
			if err != nil {
				return err
			}

			// Verify the issue belongs to the specified repository
			repo, err := be.Repository(ctx, repoName)
			if err != nil {
				return err
			}
			if issue.RepoID() != repo.ID() {
				return fmt.Errorf("issue #%d not found in repository %s", issueID, repoName)
			}

			if issue.IsOpen() {
				return fmt.Errorf("issue #%d is already open", issueID)
			}

			if err := be.ReopenIssue(ctx, issueID); err != nil {
				return err
			}

			cmd.Printf("Issue #%d reopened\n", issueID)
			return nil
		},
	}

	return cmd
}
