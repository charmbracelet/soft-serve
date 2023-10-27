package cmd

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/spf13/cobra"
)

// SettingsCommand returns a command that manages server settings.
func SettingsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "settings",
		Short: "Manage server settings",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:               "allow-keyless [true|false]",
			Short:             "Set or get allow keyless access to repositories",
			Args:              cobra.RangeArgs(0, 1),
			PersistentPreRunE: checkIfAdmin,
			RunE: func(cmd *cobra.Command, args []string) error {
				ctx := cmd.Context()
				be := backend.FromContext(ctx)
				switch len(args) {
				case 0:
					cmd.Println(be.AllowKeyless(ctx))
				case 1:
					v, _ := strconv.ParseBool(args[0])
					if err := be.SetAllowKeyless(ctx, v); err != nil {
						return err
					}
				}

				return nil
			},
		},
	)

	als := []string{access.NoAccess.String(), access.ReadOnlyAccess.String(), access.ReadWriteAccess.String(), access.AdminAccess.String()}
	cmd.AddCommand(
		&cobra.Command{
			Use:               "anon-access [ACCESS_LEVEL]",
			Short:             "Set or get the default access level for anonymous users",
			Args:              cobra.RangeArgs(0, 1),
			ValidArgs:         als,
			PersistentPreRunE: checkIfAdmin,
			RunE: func(cmd *cobra.Command, args []string) error {
				ctx := cmd.Context()
				be := backend.FromContext(ctx)
				switch len(args) {
				case 0:
					cmd.Println(be.AnonAccess(ctx))
				case 1:
					al := access.ParseAccessLevel(args[0])
					if al < 0 {
						return fmt.Errorf("invalid access level: %s. Please choose one of the following: %s", args[0], als)
					}
					if err := be.SetAnonAccess(ctx, al); err != nil {
						return err
					}
				}

				return nil
			},
		},
	)

	return cmd
}
