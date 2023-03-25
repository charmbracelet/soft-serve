package cmd

import (
	"strings"

	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/spf13/cobra"
)

func collabCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "collab",
		Aliases: []string{"collaborator", "collaborators"},
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
		Use:               "add REPOSITORY AUTHORIZED_KEY",
		Short:             "Add a collaborator to a repo",
		Args:              cobra.MinimumNArgs(2),
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := fromContext(cmd)
			repo := args[0]
			pk, c, err := backend.ParseAuthorizedKey(strings.Join(args[1:], " "))
			if err != nil {
				return err
			}

			return cfg.Backend.AddCollaborator(pk, c, repo)
		},
	}

	return cmd
}

func collabRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "remove REPOSITORY AUTHORIZED_KEY",
		Args:              cobra.MinimumNArgs(2),
		Short:             "Remove a collaborator from a repo",
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := fromContext(cmd)
			repo := args[0]
			pk, _, err := backend.ParseAuthorizedKey(strings.Join(args[1:], " "))
			if err != nil {
				return err
			}

			return cfg.Backend.RemoveCollaborator(pk, repo)
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
			cfg, _ := fromContext(cmd)
			repo := args[0]
			collabs, err := cfg.Backend.Collaborators(repo)
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
