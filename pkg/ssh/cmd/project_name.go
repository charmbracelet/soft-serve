package cmd

import (
	"strings"

	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/spf13/cobra"
)

func projectName() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "project-name REPOSITORY [NAME]",
		Aliases: []string{"project"},
		Short:   "Set or get the project name for a repository",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			rn := strings.TrimSuffix(args[0], ".git")
			switch len(args) {
			case 1:
				if err := checkIfReadable(cmd, args); err != nil {
					return err
				}

				pn, err := be.ProjectName(ctx, rn)
				if err != nil {
					return err
				}

				cmd.Println(pn)
			default:
				if err := checkIfCollab(cmd, args); err != nil {
					return err
				}
				if err := be.SetProjectName(ctx, rn, strings.Join(args[1:], " ")); err != nil {
					return err
				}
			}

			return nil
		},
	}

	return cmd
}
