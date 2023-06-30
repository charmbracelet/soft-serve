package cmd

import (
	"github.com/charmbracelet/soft-serve/server/store"
	"github.com/spf13/cobra"
)

// createCommand is the command for creating a new repository.
func createCommand() *cobra.Command {
	var private bool
	var description string
	var projectName string
	var hidden bool

	cmd := &cobra.Command{
		Use:               "create REPOSITORY",
		Short:             "Create a new repository",
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: checkIfCollab,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be, _ := fromContext(cmd)
			name := args[0]
			if _, err := be.CreateRepository(ctx, name, store.RepositoryOptions{
				Private:     private,
				Description: description,
				ProjectName: projectName,
				Hidden:      hidden,
			}); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&private, "private", "p", false, "make the repository private")
	cmd.Flags().StringVarP(&description, "description", "d", "", "set the repository description")
	cmd.Flags().StringVarP(&projectName, "name", "n", "", "set the project name")
	cmd.Flags().BoolVarP(&hidden, "hidden", "H", false, "hide the repository from the UI")

	return cmd
}
