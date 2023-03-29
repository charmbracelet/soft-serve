package cmd

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/spf13/cobra"
)

func settingCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setting",
		Short: "Manage server settings",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:               "allow-keyless [true|false]",
			Short:             "Set or get allow keyless access to repositories",
			Args:              cobra.RangeArgs(0, 1),
			PersistentPreRunE: checkIfAdmin,
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, _ := fromContext(cmd)
				switch len(args) {
				case 0:
					cmd.Println(cfg.Backend.AllowKeyless())
				case 1:
					v, _ := strconv.ParseBool(args[0])
					if err := cfg.Backend.SetAllowKeyless(v); err != nil {
						return err
					}
				}

				return nil
			},
		},
	)

	als := []string{backend.NoAccess.String(), backend.ReadOnlyAccess.String(), backend.ReadWriteAccess.String(), backend.AdminAccess.String()}
	cmd.AddCommand(
		&cobra.Command{
			Use:               "anon-access [ACCESS_LEVEL]",
			Short:             "Set or get the default access level for anonymous users",
			Args:              cobra.RangeArgs(0, 1),
			ValidArgs:         als,
			PersistentPreRunE: checkIfAdmin,
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, _ := fromContext(cmd)
				switch len(args) {
				case 0:
					cmd.Println(cfg.Backend.AnonAccess())
				case 1:
					al := backend.ParseAccessLevel(args[0])
					if al < 0 {
						return fmt.Errorf("invalid access level: %s. Please choose one of the following: %s", args[0], als)
					}
					if err := cfg.Backend.SetAnonAccess(al); err != nil {
						return err
					}
				}

				return nil
			},
		},
	)

	return cmd
}
