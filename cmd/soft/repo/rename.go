package repo

import (
	"github.com/charmbracelet/soft-serve/cmd"
	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/spf13/cobra"
)

func renameCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "rename REPOSITORY NEW_NAME",
		Aliases: []string{"mv", "move"},
		Short:   "Rename an existing repository",
		Args:    cobra.ExactArgs(2),
		RunE: func(co *cobra.Command, args []string) error {
			ctx := co.Context()
			be := backend.FromContext(ctx)
			oldName := args[0]
			newName := args[1]

			if !cmd.CheckUserHasAccess(co, oldName, access.ReadWriteAccess) {
				return proto.ErrUnauthorized
			}

			return be.RenameRepository(ctx, oldName, newName)
		},
	}

	return cmd
}
