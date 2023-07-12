package cmd

import "github.com/spf13/cobra"

func renameCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "rename REPOSITORY NEW_NAME",
		Aliases:           []string{"mv", "move"},
		Short:             "Rename an existing repository",
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: checkIfCollab,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			_, be, _ := fromContext(cmd)
			oldName := args[0]
			newName := args[1]
			if err := be.RenameRepository(ctx, oldName, newName); err != nil {
				return err
			}
			return nil
		},
	}

	return cmd
}
