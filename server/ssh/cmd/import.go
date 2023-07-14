package cmd

import (
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/proto"
	"github.com/spf13/cobra"
)

// importCommand is the command for creating a new repository.
func importCommand() *cobra.Command {
	var private bool
	var description string
	var projectName string
	var mirror bool
	var hidden bool

	cmd := &cobra.Command{
		Use:               "import REPOSITORY REMOTE",
		Short:             "Import a new repository from remote",
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: checkIfCollab,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			name := args[0]
			remote := args[1]
			if _, err := be.ImportRepository(ctx, name, remote, proto.RepositoryOptions{
				Private:     private,
				Description: description,
				ProjectName: projectName,
				Mirror:      mirror,
				Hidden:      hidden,
			}); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&mirror, "mirror", "m", false, "mirror the repository")
	cmd.Flags().BoolVarP(&private, "private", "p", false, "make the repository private")
	cmd.Flags().StringVarP(&description, "description", "d", "", "set the repository description")
	cmd.Flags().StringVarP(&projectName, "name", "n", "", "set the project name")
	cmd.Flags().BoolVarP(&hidden, "hidden", "H", false, "hide the repository from the UI")

	return cmd
}
