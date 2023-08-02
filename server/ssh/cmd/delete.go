package cmd

import (
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/spf13/cobra"
)

func deleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "delete REPOSITORY",
		Aliases:           []string{"del", "remove", "rm"},
		Short:             "Delete a repository",
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: checkIfCollab,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			name := args[0]

			return be.DeleteRepository(ctx, name)
		},
	}

	return cmd
}
