package cmd

import (
	"errors"

	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/task"
	"github.com/spf13/cobra"
)

// importCommand is the command for creating a new repository.
func importCommand() *cobra.Command {
	var private bool
	var description string
	var projectName string
	var mirror bool
	var hidden bool
	var lfs bool
	var lfsEndpoint string

	cmd := &cobra.Command{
		Use:               "import REPOSITORY REMOTE",
		Short:             "Import a new repository from remote",
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: checkIfCollab,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			user := proto.UserFromContext(ctx)
			name := args[0]
			remote := args[1]
			if _, err := be.ImportRepository(ctx, name, user, remote, proto.RepositoryOptions{
				Private:     private,
				Description: description,
				ProjectName: projectName,
				Mirror:      mirror,
				Hidden:      hidden,
				LFS:         lfs,
				LFSEndpoint: lfsEndpoint,
			}); err != nil {
				if errors.Is(err, task.ErrAlreadyStarted) {
					return errors.New("import already in progress")
				}

				return err
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&lfs, "lfs", "", false, "pull Git LFS objects")
	cmd.Flags().StringVarP(&lfsEndpoint, "lfs-endpoint", "", "", "set the Git LFS endpoint")
	cmd.Flags().BoolVarP(&mirror, "mirror", "m", false, "mirror the repository")
	cmd.Flags().BoolVarP(&private, "private", "p", false, "make the repository private")
	cmd.Flags().StringVarP(&description, "description", "d", "", "set the repository description")
	cmd.Flags().StringVarP(&projectName, "name", "n", "", "set the project name")
	cmd.Flags().BoolVarP(&hidden, "hidden", "H", false, "hide the repository from the UI")

	return cmd
}
