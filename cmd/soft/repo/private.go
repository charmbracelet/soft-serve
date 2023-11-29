package repo

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/soft-serve/cmd"
	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/spf13/cobra"
)

func privateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "private REPOSITORY [true|false]",
		Short: "Set or get a repository private property",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(co *cobra.Command, args []string) error {
			ctx := co.Context()
			be := backend.FromContext(ctx)
			rn := strings.TrimSuffix(args[0], ".git")

			switch len(args) {
			case 1:
				if !cmd.CheckUserHasAccess(co, rn, access.ReadOnlyAccess) {
					return proto.ErrUnauthorized
				}

				isPrivate, err := be.IsPrivate(ctx, rn)
				if err != nil {
					return err
				}

				co.Println(isPrivate)
			case 2:
				if !cmd.CheckUserHasAccess(co, rn, access.ReadWriteAccess) {
					return proto.ErrUnauthorized
				}

				isPrivate, err := strconv.ParseBool(args[1])
				if err != nil {
					return err
				}
				if err := be.SetPrivate(ctx, rn, isPrivate); err != nil {
					return err
				}
			}
			return nil
		},
	}

	return cmd
}
