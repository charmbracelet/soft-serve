package settings

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/soft-serve/cmd"
	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/spf13/cobra"
)

var (
	// Command returns a command that manages server settings.
	Command = &cobra.Command{
		Use:                "settings",
		Short:              "Manage server settings",
		PersistentPreRunE:  cmd.InitBackendContext,
		PersistentPostRunE: cmd.CloseDBContext,
	}
)

func init() {
	Command.AddCommand(
		&cobra.Command{
			Use:   "allow-keyless [true|false]",
			Short: "Set or get allow keyless access to repositories",
			Args:  cobra.RangeArgs(0, 1),
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

	als := []string{
		access.NoAccess.String(),
		access.ReadOnlyAccess.String(),
		access.ReadWriteAccess.String(),
		access.AdminAccess.String(),
	}
	Command.AddCommand(
		&cobra.Command{
			Use:       "anon-access [ACCESS_LEVEL]",
			Short:     "Set or get the default access level for anonymous users",
			Args:      cobra.RangeArgs(0, 1),
			ValidArgs: als,
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
}
