package cmd

import (
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/spf13/cobra"
)

func hiddenCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "hidden REPOSITORY [TRUE|FALSE]",
		Short:             "Hide or unhide a repository",
		Aliases:           []string{"hide"},
		Args:              cobra.MinimumNArgs(1),
		PersistentPreRunE: checkIfReadable,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repo := args[0]
			switch len(args) {
			case 1:
				hidden, err := be.IsHidden(ctx, repo)
				if err != nil {
					return err
				}

				cmd.Println(hidden)
			case 2:
				if err := checkIfCollab(cmd, args); err != nil {
					return err
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
