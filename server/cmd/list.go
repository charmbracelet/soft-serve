package cmd

import (
	"github.com/charmbracelet/soft-serve/server/access"
	"github.com/charmbracelet/soft-serve/server/auth"
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
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be, s := fromContext(cmd)
			repos, err := be.Repositories(ctx, 1, 10)
			if err != nil {
				return err
			}

			user, err := be.Authenticate(ctx, auth.NewPublicKey(s.PublicKey()))
			if err != nil {
				return err
			}

			for _, r := range repos {
				ac, err := be.AccessLevel(ctx, r.Name(), user)
				if err != nil {
					continue
				}

				if ac >= access.ReadOnlyAccess {
					if !r.IsHidden() || all {
						cmd.Println(r.Name())
					}
				}
			}
			return nil
		},
	}

	listCmd.Flags().BoolVarP(&all, "all", "a", false, "List all repositories")

	return listCmd
}
