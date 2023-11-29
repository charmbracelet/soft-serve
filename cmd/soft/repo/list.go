package repo

import (
	"github.com/charmbracelet/soft-serve/cmd"
	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/spf13/cobra"
)

// listCommand returns a command that list file or directory at path.
func listCommand() *cobra.Command {
	var all bool

	listCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List repositories",
		Args:    cobra.NoArgs,
		RunE: func(co *cobra.Command, _ []string) error {
			ctx := co.Context()
			be := backend.FromContext(ctx)
			repos, err := be.Repositories(ctx)
			if err != nil {
				return err
			}

			for _, r := range repos {
				if !cmd.CheckUserHasAccess(co, r.Name(), access.ReadOnlyAccess) {
					continue
				}

				if !r.IsHidden() || all {
					co.Println(r.Name())
				}
			}
			return nil
		},
	}

	listCmd.Flags().BoolVarP(&all, "all", "a", false, "List all repositories")

	return listCmd
}
