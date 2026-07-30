package cmd

import (
	"context"
	"errors"

	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/spf13/cobra"
)

func collabCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "collab",
		Aliases: []string{"collabs", "collaborator", "collaborators"},
		Short:   "Manage collaborators",
	}

	cmd.AddCommand(
		collabAddCommand(),
		collabRemoveCommand(),
		collabListCommand(),
	)

	return cmd
}

// checkCollabGrant reports whether the caller may set a collaborator's access
// level on a repository to level.
//
// Server admins may grant anything. Everyone else is bounded by their own
// access level on the repository, so a read-write collaborator cannot mint an
// admin-access collaborator and escalate beyond their own permissions.
func checkCollabGrant(ctx context.Context, repo string, level access.AccessLevel) error {
	if isServerAdmin(ctx) {
		return nil
	}

	if proto.UserFromContext(ctx) == nil {
		return proto.ErrUnauthorized
	}

	caller := repoAccessLevel(ctx, repo)
	if level > caller {
		return proto.ErrExceedsAccessLevel
	}

	return nil
}

// checkCollabDemote reports whether the caller may remove or overwrite an
// existing collaborator on a repository.
//
// Removal is a privileged change in the same way granting is: without this,
// a read-write collaborator could remove an admin-access collaborator and
// then re-add them at a lower level, demoting someone above them.
func checkCollabDemote(ctx context.Context, repo string, username string) error {
	if isServerAdmin(ctx) {
		return nil
	}

	if proto.UserFromContext(ctx) == nil {
		return proto.ErrUnauthorized
	}

	be := backend.FromContext(ctx)
	current, isCollab, err := be.IsCollaborator(ctx, repo, username)
	if err != nil {
		// A missing row just means the user is not a collaborator yet, which
		// is the common case when adding one. Any other error is real and
		// must fail closed.
		if !errors.Is(err, db.ErrRecordNotFound) {
			return err
		}
		return nil
	}

	if !isCollab {
		return nil
	}

	if current > repoAccessLevel(ctx, repo) {
		return proto.ErrExceedsAccessLevel
	}

	return nil
}

func collabAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "add REPOSITORY USERNAME [LEVEL]",
		Short:             "Add a collaborator to a repo",
		Long:              "Add a collaborator to a repo. LEVEL can be one of: no-access, read-only, read-write, or admin-access. Defaults to read-write.",
		Args:              cobra.RangeArgs(2, 3),
		PersistentPreRunE: checkIfReadableAndCollab,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repo := repoArg(args)
			username := args[1]
			level := access.ReadWriteAccess
			if len(args) > 2 {
				level = access.ParseAccessLevel(args[2])
				if level < 0 {
					return access.ErrInvalidAccessLevel
				}
			}

			if err := checkCollabGrant(ctx, repo, level); err != nil {
				return err
			}

			if err := checkCollabDemote(ctx, repo, username); err != nil {
				return err
			}

			return be.AddCollaborator(ctx, repo, username, level)
		},
	}

	return cmd
}

func collabRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "remove REPOSITORY USERNAME",
		Args:              cobra.ExactArgs(2),
		Short:             "Remove a collaborator from a repo",
		PersistentPreRunE: checkIfReadableAndCollab,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repo := repoArg(args)
			username := args[1]

			if err := checkCollabDemote(ctx, repo, username); err != nil {
				return err
			}

			return be.RemoveCollaborator(ctx, repo, username)
		},
	}

	return cmd
}

func collabListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "list REPOSITORY",
		Short:             "List collaborators for a repo",
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: checkIfReadableAndCollab,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repo := repoArg(args)
			collabs, err := be.Collaborators(ctx, repo)
			if err != nil {
				return err
			}

			for _, c := range collabs {
				cmd.Println(c)
			}

			return nil
		},
	}

	return cmd
}
