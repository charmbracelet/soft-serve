package repo

import (
	"github.com/charmbracelet/soft-serve/cmd"
	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/spf13/cobra"
)

func deleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete REPOSITORY",
		Aliases: []string{"del", "remove", "rm"},
		Short:   "Delete a repository",
		Args:    cobra.ExactArgs(1),
		RunE: func(co *cobra.Command, args []string) error {
			ctx := co.Context()
			be := backend.FromContext(ctx)
			name := args[0]
			if !cmd.CheckUserHasAccess(co, name, access.ReadWriteAccess) {
				return proto.ErrUnauthorized
			}

			return be.DeleteRepository(ctx, name)
		},
	}

	return cmd
}
