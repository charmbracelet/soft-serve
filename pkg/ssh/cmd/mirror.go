package cmd

import (
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/spf13/cobra"
)

func mirrorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "mirror REPOSITORY [true|false]",
		Short:             "Set or get a repository mirror property",
		Args:              cobra.RangeArgs(1, 2),
		PersistentPreRunE: checkIfReadable,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			rn := args[0]

			switch len(args) {
			case 1:
				isMirror, err := be.IsMirror(ctx, rn)
				if err != nil {
					return err
				}

				cmd.Println(isMirror)
			case 2:
				if err := checkIfRepoCollab(cmd, args); err != nil {
					return err
				}

				isMirror := args[1] == "true"
				if err := be.SetMirror(ctx, rn, isMirror); err != nil {
					return err
				}
			}

			return nil
		},
	}

	return cmd
}
