package repo

import (
	"github.com/charmbracelet/soft-serve/cmd"
	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
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

func collabAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add REPOSITORY USERNAME [LEVEL]",
		Short: "Add a collaborator to a repo",
		Long:  "Add a collaborator to a repo. LEVEL can be one of: no-access, read-only, read-write, or admin-access. Defaults to read-write.",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(c *cobra.Command, args []string) error {
			ctx := c.Context()
			be := backend.FromContext(ctx)
			repo := args[0]
			rr, err := be.Repository(ctx, repo)
			if err != nil {
				return err
			}

			if !cmd.CheckUserHasAccess(c, rr.Name(), access.ReadWriteAccess) {
				return proto.ErrUnauthorized
			}

			username := args[1]
			level := access.ReadWriteAccess
			if len(args) > 2 {
				level = access.ParseAccessLevel(args[2])
				if level < 0 {
					return access.ErrInvalidAccessLevel
				}
			}

			return be.AddCollaborator(ctx, repo, username, level)
		},
	}

	return cmd
}

func collabRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove REPOSITORY USERNAME",
		Args:  cobra.ExactArgs(2),
		Short: "Remove a collaborator from a repo",
		RunE: func(c *cobra.Command, args []string) error {
			ctx := c.Context()
			be := backend.FromContext(ctx)
			repo := args[0]
			rr, err := be.Repository(ctx, repo)
			if err != nil {
				return err
			}

			if !cmd.CheckUserHasAccess(c, rr.Name(), access.ReadWriteAccess) {
				return proto.ErrUnauthorized
			}

			username := args[1]

			return be.RemoveCollaborator(ctx, repo, username)
		},
	}

	return cmd
}

func collabListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list REPOSITORY",
		Short: "List collaborators for a repo",
		Args:  cobra.ExactArgs(1),
		RunE: func(co *cobra.Command, args []string) error {
			ctx := co.Context()
			be := backend.FromContext(ctx)
			repo := args[0]
			rr, err := be.Repository(ctx, repo)
			if err != nil {
				return err
			}

			if !cmd.CheckUserHasAccess(co, rr.Name(), access.ReadWriteAccess) {
				return proto.ErrUnauthorized
			}

			collabs, err := be.Collaborators(ctx, repo)
			if err != nil {
				return err
			}

			for _, c := range collabs {
				co.Println(c)
			}

			return nil
		},
	}

	return cmd
}
