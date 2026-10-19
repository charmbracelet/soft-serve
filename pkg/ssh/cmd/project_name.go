package cmd

import (
	"strings"

	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/spf13/cobra"
)

func projectName() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "project-name REPOSITORY [NAME]",
		Aliases:           []string{"project"},
		Short:             "Set or get the project name for a repository",
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
				cmd.Println(rr.ProjectName())
			default:
				if err := requireLevel(ctx, access.ReadWriteAccess); err != nil {
					return err
				}
				if err := be.SetProjectName(ctx, rr.Name(), strings.Join(args[1:], " ")); err != nil {
					return err
				}
			}

			return nil
		},
	}

	return cmd
}
