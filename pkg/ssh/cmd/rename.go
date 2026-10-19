package cmd

import (
	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/utils"
	"github.com/spf13/cobra"
)

func renameCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "rename REPOSITORY NEW_NAME",
		Aliases:           []string{"mv", "move"},
		Short:             "Rename an existing repository",
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: requireRepo(access.ReadWriteAccess),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			oldName := repoArg(args)
			newName := utils.SanitizeRepo(args[1])

			// The destination is gated like create: it may already be
			// somebody else's repository.
			if repoAccessLevel(ctx, newName) < access.ReadWriteAccess {
				return proto.ErrUnauthorized
			}

			return be.RenameRepository(ctx, oldName, newName)
		},
	}

	return cmd
}
