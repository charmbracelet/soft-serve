package repo

import (
	"strings"

	"github.com/charmbracelet/soft-serve/cmd"
	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/spf13/cobra"
)

func descriptionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "description REPOSITORY [DESCRIPTION]",
		Aliases: []string{"desc"},
		Short:   "Set or get the description for a repository",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(co *cobra.Command, args []string) error {
			ctx := co.Context()
			be := backend.FromContext(ctx)
			rn := strings.TrimSuffix(args[0], ".git")
			switch len(args) {
			case 1:
				if !cmd.CheckUserHasAccess(co, rn, access.ReadOnlyAccess) {
					return proto.ErrUnauthorized
				}

				desc, err := be.Description(ctx, rn)
				if err != nil {
					return err
				}

				co.Println(desc)
			default:
				if !cmd.CheckUserHasAccess(co, rn, access.ReadWriteAccess) {
					return proto.ErrUnauthorized
				}

				if err := be.SetDescription(ctx, rn, strings.Join(args[1:], " ")); err != nil {
					return err
				}
			}

			return nil
		},
	}

	return cmd
}
