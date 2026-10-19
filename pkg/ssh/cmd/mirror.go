package cmd

import (
	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/spf13/cobra"
)

func mirrorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "mirror REPOSITORY [true|false]",
		Short:             "Set or get a repository mirror property",
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
				cmd.Println(rr.IsMirror())
			case 2:
				if err := requireLevel(ctx, access.ReadWriteAccess); err != nil {
					return err
				}

				isMirror := args[1] == "true"
				if err := be.SetMirror(ctx, rr.Name(), isMirror); err != nil {
					return err
				}
			}

			return nil
		},
	}

	return cmd
}
