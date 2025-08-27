package cmd

import (
	"fmt"

	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/proto"
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
			cfg := config.FromContext(ctx)
			be := backend.FromContext(ctx)
			user := proto.UserFromContext(ctx)
			name := args[0]
			r, err := be.CreateRepository(ctx, name, user, proto.RepositoryOptions{
				Private:     private,
				Description: description,
				ProjectName: projectName,
				Hidden:      hidden,
			})
			if err != nil {
				return err //nolint:wrapcheck
			}

			cloneurl := fmt.Sprintf("%s/%s.git", cfg.SSH.PublicURL, r.Name())
			cmd.PrintErrf("Created repository %s\n", r.Name())
			cmd.Println(cloneurl)

			return nil
		},
	}

	cmd.Flags().BoolVarP(&private, "private", "p", false, "make the repository private")
	cmd.Flags().StringVarP(&description, "description", "d", "", "set the repository description")
	cmd.Flags().StringVarP(&projectName, "name", "n", "", "set the project name")
	cmd.Flags().BoolVarP(&hidden, "hidden", "H", false, "hide the repository from the UI")

	return cmd
}
