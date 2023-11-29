package repo

import (
	"github.com/charmbracelet/soft-serve/cmd"
	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/spf13/cobra"
)

func hiddenCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "hidden REPOSITORY [TRUE|FALSE]",
		Short:   "Hide or unhide a repository",
		Aliases: []string{"hide"},
		Args:    cobra.MinimumNArgs(1),
		RunE: func(co *cobra.Command, args []string) error {
			ctx := co.Context()
			be := backend.FromContext(ctx)
			repo := args[0]
			switch len(args) {
			case 1:
				if !cmd.CheckUserHasAccess(co, repo, access.ReadOnlyAccess) {
					return proto.ErrUnauthorized
				}

				hidden, err := be.IsHidden(ctx, repo)
				if err != nil {
					return err
				}

				co.Println(hidden)
			case 2:
				if !cmd.CheckUserHasAccess(co, repo, access.ReadWriteAccess) {
					return proto.ErrUnauthorized
				}

				hidden := args[1] == "true"
				if err := be.SetHidden(ctx, repo, hidden); err != nil {
					return err
				}
			}

			return nil
		},
	}

	return cmd
}
