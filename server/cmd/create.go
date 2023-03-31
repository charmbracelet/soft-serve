package cmd

import (
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/spf13/cobra"
)

// createCommand is the command for creating a new repository.
func createCommand() *cobra.Command {
	var private bool
	var description string
	var mirror string
	var projectName string

	cmd := &cobra.Command{
		Use:               "create REPOSITORY",
		Short:             "Create a new repository",
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := fromContext(cmd)
			name := args[0]
			if _, err := cfg.Backend.CreateRepository(name, backend.RepositoryOptions{
				Private:     private,
				Mirror:      mirror,
				Description: description,
				ProjectName: projectName,
			}); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&private, "private", "p", false, "make the repository private")
	cmd.Flags().StringVarP(&description, "description", "d", "", "set the repository description")
	cmd.Flags().StringVarP(&mirror, "mirror", "m", "", "set the mirror repository")
	cmd.Flags().StringVarP(&projectName, "name", "n", "", "set the project name")

	return cmd
}
