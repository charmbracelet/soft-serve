package cmd

import (
	"strings"

	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/spf13/cobra"
)

func descriptionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "description REPOSITORY [DESCRIPTION]",
		Aliases:           []string{"desc"},
		Short:             "Set or get the description for a repository",
		Args:              cobra.MinimumNArgs(1),
		PersistentPreRunE: checkIfReadable,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			rn := strings.TrimSuffix(args[0], ".git")
			switch len(args) {
			case 1:
				desc, err := be.Description(ctx, rn)
				if err != nil {
					return err
				}

				cmd.Println(desc)
			default:
				if err := checkIfCollab(cmd, args); err != nil {
					return err
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
