package cmd

import "github.com/spf13/cobra"

// createCommand is the command for creating a new repository.
func createCommand() *cobra.Command {
	var private bool
	var description string
	cmd := &cobra.Command{
		Use:               "create REPOSITORY",
		Short:             "Create a new repository",
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := fromContext(cmd)
			name := args[0]
			if _, err := cfg.Backend.CreateRepository(name, private); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&private, "private", "p", false, "make the repository private")
	cmd.Flags().StringVarP(&description, "description", "d", "", "set the repository description")
	return cmd
}
