package cmd

import (
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/spf13/cobra"
)

func renameCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "rename REPOSITORY NEW_NAME",
		Aliases:           []string{"mv", "move"},
		Short:             "Rename an existing repository",
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: checkIfCollab,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			oldName := args[0]
			newName := args[1]

			return be.RenameRepository(ctx, oldName, newName)
		},
	}

	return cmd
}
