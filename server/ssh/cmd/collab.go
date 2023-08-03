package cmd

import (
	"github.com/charmbracelet/soft-serve/server/access"
	"github.com/charmbracelet/soft-serve/server/backend"
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
		Use:               "add REPOSITORY USERNAME [LEVEL]",
		Short:             "Add a collaborator to a repo",
		Long:              "Add a collaborator to a repo. LEVEL can be one of: no-access, read-only, read-write, or admin-access. Defaults to read-write.",
		Args:              cobra.RangeArgs(2, 3),
		PersistentPreRunE: checkIfCollab,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repo := args[0]
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
		Use:               "remove REPOSITORY USERNAME",
		Args:              cobra.ExactArgs(2),
		Short:             "Remove a collaborator from a repo",
		PersistentPreRunE: checkIfCollab,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repo := args[0]
			username := args[1]

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
		PersistentPreRunE: checkIfCollab,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repo := args[0]
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
