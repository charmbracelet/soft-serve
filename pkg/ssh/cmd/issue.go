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
		issueEditCommand(),
		issueDeleteCommand(),
		issueCommentCommand(),
		issueLabelCommand(),
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
		PersistentPreRunE: checkIfReadable,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repoName := args[0]

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
	var labelName string

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

			var (
				issues []proto.Issue
				err    error
			)
			if cmd.Flags().Changed("label") {
				issues, err = be.GetIssuesByRepositoryAndLabel(ctx, repoName, labelName, status)
			} else {
				issues, err = be.GetIssuesByRepository(ctx, repoName, status)
			}
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

	cmd.Flags().StringVarP(&status, "status", "s", "open", "Filter by status (open, closed, or all)")
	cmd.Flags().StringVarP(&labelName, "label", "l", "", "Filter by label name")

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

			issue, repo, err := getIssueAndVerifyRepo(cmd, be, repoName, issueID)
			if err != nil {
				return err
			}
			_ = repo

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
			labels, _ := be.GetIssueLabels(ctx, issue.ID())
			if len(labels) > 0 {
				names := make([]string, len(labels))
				for i, l := range labels {
					names[i] = l.Name()
				}
				cmd.Printf("Labels: %s\n", strings.Join(names, ", "))
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

			issue, _, err := getIssueAndVerifyRepo(cmd, be, repoName, issueID)
			if err != nil {
				return err
			}

			if issue.IsClosed() {
				return fmt.Errorf("issue #%d is already closed", issueID)
			}

			user := proto.UserFromContext(ctx)
			if user == nil {
				return fmt.Errorf("user not found")
			}

			if err := be.CloseIssue(ctx, issueID, issue.RepoID(), user.ID()); err != nil {
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

			issue, _, err := getIssueAndVerifyRepo(cmd, be, repoName, issueID)
			if err != nil {
				return err
			}

			if issue.IsOpen() {
				return fmt.Errorf("issue #%d is already open", issueID)
			}

			if err := be.ReopenIssue(ctx, issueID, issue.RepoID()); err != nil {
				return err
			}

			cmd.Printf("Issue #%d reopened\n", issueID)
			return nil
		},
	}

	return cmd
}

// issueEditCommand returns a command for editing an issue.
func issueEditCommand() *cobra.Command {
	var title string
	var body string

	cmd := &cobra.Command{
		Use:               "edit REPOSITORY ISSUE_ID",
		Short:             "Edit an issue title or body",
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

			issue, _, err := getIssueAndVerifyRepo(cmd, be, repoName, issueID)
			if err != nil {
				return err
			}

			// Only the issue author or an admin may edit.
			user := proto.UserFromContext(ctx)
			if user == nil {
				return fmt.Errorf("user not found")
			}
			if user.ID() != issue.UserID() {
				if be.AccessLevelForUser(ctx, repoName, user) < access.AdminAccess {
					return fmt.Errorf("permission denied: only the issue author or an admin can edit this issue")
				}
			}

			if !cmd.Flags().Changed("title") {
				title = issue.Title()
			}

			var bodyPtr *string
			if cmd.Flags().Changed("body") {
				bodyPtr = &body
			}

			if err := be.UpdateIssue(ctx, issueID, issue.RepoID(), title, bodyPtr); err != nil {
				return err
			}

			cmd.Printf("Issue #%d updated\n", issueID)
			return nil
		},
	}

	cmd.Flags().StringVarP(&title, "title", "t", "", "New issue title")
	cmd.Flags().StringVarP(&body, "body", "b", "", "New issue body")

	return cmd
}

// issueDeleteCommand returns a command for deleting an issue.
func issueDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "delete REPOSITORY ISSUE_ID",
		Short:             "Delete an issue",
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

			issue, _, err := getIssueAndVerifyRepo(cmd, be, repoName, issueID)
			if err != nil {
				return err
			}

			// Only the issue author or an admin may delete.
			user := proto.UserFromContext(ctx)
			if user == nil {
				return fmt.Errorf("user not found")
			}
			if user.ID() != issue.UserID() {
				if be.AccessLevelForUser(ctx, repoName, user) < access.AdminAccess {
					return fmt.Errorf("permission denied: only the issue author or an admin can delete this issue")
				}
			}

			if err := be.DeleteIssue(ctx, issueID, issue.RepoID()); err != nil {
				return err
			}

			cmd.Printf("Issue #%d deleted\n", issueID)
			return nil
		},
	}

	return cmd
}

// issueCommentCommand returns a command for managing issue comments.
func issueCommentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comment",
		Short: "Manage issue comments",
	}

	cmd.AddCommand(
		issueCommentAddCommand(),
		issueCommentListCommand(),
		issueCommentEditCommand(),
		issueCommentDeleteCommand(),
	)

	return cmd
}

// issueCommentAddCommand adds a comment to an issue.
func issueCommentAddCommand() *cobra.Command {
	var body string

	cmd := &cobra.Command{
		Use:               "add REPOSITORY ISSUE_ID",
		Short:             "Add a comment to an issue",
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

			issue, _, err := getIssueAndVerifyRepo(cmd, be, repoName, issueID)
			if err != nil {
				return err
			}

			user := proto.UserFromContext(ctx)
			if user == nil {
				return fmt.Errorf("user not found")
			}

			comment, err := be.AddIssueComment(ctx, issue.ID(), user.ID(), body)
			if err != nil {
				return err
			}

			cmd.Printf("Comment #%d added to issue #%d\n", comment.ID(), issue.ID())
			return nil
		},
	}

	cmd.Flags().StringVarP(&body, "body", "b", "", "Comment body (required)")
	_ = cmd.MarkFlagRequired("body")

	return cmd
}

// issueCommentListCommand lists all comments on an issue.
func issueCommentListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "list REPOSITORY ISSUE_ID",
		Aliases:           []string{"ls"},
		Short:             "List comments on an issue",
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

			issue, _, err := getIssueAndVerifyRepo(cmd, be, repoName, issueID)
			if err != nil {
				return err
			}

			comments, err := be.GetIssueComments(ctx, issue.ID())
			if err != nil {
				return err
			}

			if len(comments) == 0 {
				cmd.Println("No comments found")
				return nil
			}

			for _, c := range comments {
				author, _ := be.UserByID(ctx, c.UserID())
				authorName := "unknown"
				if author != nil {
					authorName = author.Username()
				}
				cmd.Printf("Comment #%d by %s on %s\n", c.ID(), authorName, c.CreatedAt().Format("2006-01-02 15:04:05"))
				cmd.Println(c.Body())
				cmd.Println()
			}
			return nil
		},
	}

	return cmd
}

// issueCommentEditCommand edits the body of a comment.
func issueCommentEditCommand() *cobra.Command {
	var body string

	cmd := &cobra.Command{
		Use:               "edit REPOSITORY ISSUE_ID COMMENT_ID",
		Short:             "Edit a comment",
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
			commentID, err := strconv.ParseInt(args[2], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid comment ID: %s", args[2])
			}

			issue, _, err := getIssueAndVerifyRepo(cmd, be, repoName, issueID)
			if err != nil {
				return err
			}

			comment, err := getCommentAndVerifyIssue(be, cmd, issue.ID(), commentID)
			if err != nil {
				return err
			}

			user := proto.UserFromContext(ctx)
			if user == nil {
				return fmt.Errorf("user not found")
			}
			if user.ID() != comment.UserID() {
				if be.AccessLevelForUser(ctx, repoName, user) < access.AdminAccess {
					return fmt.Errorf("permission denied: only the comment author or an admin can edit this comment")
				}
			}

			if err := be.UpdateIssueComment(ctx, commentID, body); err != nil {
				return err
			}

			cmd.Printf("Comment #%d updated\n", commentID)
			return nil
		},
	}

	cmd.Flags().StringVarP(&body, "body", "b", "", "New comment body (required)")
	_ = cmd.MarkFlagRequired("body")

	return cmd
}

// issueCommentDeleteCommand deletes a comment.
func issueCommentDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "delete REPOSITORY ISSUE_ID COMMENT_ID",
		Short:             "Delete a comment",
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
			commentID, err := strconv.ParseInt(args[2], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid comment ID: %s", args[2])
			}

			issue, _, err := getIssueAndVerifyRepo(cmd, be, repoName, issueID)
			if err != nil {
				return err
			}

			comment, err := getCommentAndVerifyIssue(be, cmd, issue.ID(), commentID)
			if err != nil {
				return err
			}

			user := proto.UserFromContext(ctx)
			if user == nil {
				return fmt.Errorf("user not found")
			}
			if user.ID() != comment.UserID() {
				if be.AccessLevelForUser(ctx, repoName, user) < access.AdminAccess {
					return fmt.Errorf("permission denied: only the comment author or an admin can delete this comment")
				}
			}

			if err := be.DeleteIssueComment(ctx, commentID); err != nil {
				return err
			}

			cmd.Printf("Comment #%d deleted\n", commentID)
			return nil
		},
	}

	return cmd
}

// getCommentAndVerifyIssue fetches a comment and verifies it belongs to the given issue.
// Returns a uniform "not found" error to avoid leaking global comment IDs.
func getCommentAndVerifyIssue(be *backend.Backend, cmd *cobra.Command, issueID, commentID int64) (proto.IssueComment, error) {
	comment, err := be.GetIssueComment(cmd.Context(), commentID)
	if err != nil || comment.IssueID() != issueID {
		return nil, fmt.Errorf("comment #%d not found in issue #%d", commentID, issueID)
	}
	return comment, nil
}

// getIssueAndVerifyRepo fetches an issue and verifies it belongs to the given repository.
// Returns a uniform "not found" error regardless of whether the issue doesn't exist
// or simply belongs to a different repo, to avoid leaking global issue IDs.
func getIssueAndVerifyRepo(cmd *cobra.Command, be *backend.Backend, repoName string, issueID int64) (proto.Issue, proto.Repository, error) {
	ctx := cmd.Context()

	repo, err := be.Repository(ctx, repoName)
	if err != nil {
		return nil, nil, err
	}

	issue, err := be.GetIssue(ctx, issueID)
	if err != nil || issue.RepoID() != repo.ID() {
		return nil, nil, fmt.Errorf("issue #%d not found in repository %s", issueID, repoName)
	}

	return issue, repo, nil
}
