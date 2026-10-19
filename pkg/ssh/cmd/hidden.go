package cmd

import (
	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/spf13/cobra"
)

func hiddenCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "hidden REPOSITORY [TRUE|FALSE]",
		Short:             "Hide or unhide a repository",
		Aliases:           []string{"hide"},
		Args:              cobra.MinimumNArgs(1),
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
				cmd.Println(rr.IsHidden())
			case 2:
				if err := requireLevel(ctx, access.ReadWriteAccess); err != nil {
					return err
				}

				hidden := args[1] == "true"
				if err := be.SetHidden(ctx, rr.Name(), hidden); err != nil {
					return err
				}
			}

			return nil
		},
	}

	return cmd
}
