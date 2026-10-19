package cmd

import (
	"strconv"

	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/spf13/cobra"
)

func privateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "private REPOSITORY [true|false]",
		Short:             "Set or get a repository private property",
		Args:              cobra.RangeArgs(1, 2),
		PersistentPreRunE: requireRepo(access.ReadOnlyAccess),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			rr, _, err := repoFromContext(ctx)
			if err != nil {
				return err
			}

			switch len(args) {
			case 1:
				cmd.Println(rr.IsPrivate())
			case 2:
				isPrivate, err := strconv.ParseBool(args[1])
				if err != nil {
					return err
				}
				if err := requireLevel(ctx, access.ReadWriteAccess); err != nil {
					return err
				}
				if err := be.SetPrivate(ctx, rr.Name(), isPrivate); err != nil {
					return err
				}
			}
			return nil
		},
	}

	return cmd
}
