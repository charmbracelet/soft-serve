package cmd

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/spf13/cobra"
)

// SettingsCommand returns a command that manages server settings.
func SettingsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "settings",
		Short: "Manage server settings",
		// Gate the whole command tree rather than each subcommand. Cobra
		// runs the nearest persistent pre-run found when walking up from
		// the invoked command, so subcommands added later are gated by
		// default instead of silently unprotected.
		PersistentPreRunE: checkIfServerAdmin,
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "allow-keyless [true|false]",
			Short: "Set or get allow keyless access to repositories",
			Args:  cobra.RangeArgs(0, 1),
			RunE: func(cmd *cobra.Command, args []string) error {
				ctx := cmd.Context()
				be := backend.FromContext(ctx)
				cfg := config.FromContext(ctx)
				switch len(args) {
				case 0:
					cmd.Println(be.AllowKeyless(ctx))
				case 1:
					v, _ := strconv.ParseBool(args[0])
					if err := be.SetAllowKeyless(ctx, v); err != nil {
						return err
					}
					warnIfAllowKeylessOverridden(cmd, cfg)
				}

				return nil
			},
		},
	)

	als := []string{access.NoAccess.String(), access.ReadOnlyAccess.String(), access.ReadWriteAccess.String(), access.AdminAccess.String()}
	cmd.AddCommand(
		&cobra.Command{
			Use:       "anon-access [ACCESS_LEVEL]",
			Short:     "Set or get the default access level for anonymous users",
			Args:      cobra.RangeArgs(0, 1),
			ValidArgs: als,
			RunE: func(cmd *cobra.Command, args []string) error {
				ctx := cmd.Context()
				be := backend.FromContext(ctx)
				cfg := config.FromContext(ctx)
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
					warnIfAnonAccessOverridden(cmd, cfg)
				}

				return nil
			},
		},
	)

	return cmd
}

// warnIfAllowKeylessOverridden warns on the command's stderr if a server
// config override is masking the allow-keyless value that was just written
// to the database. Without this, an admin changing the setting via this
// command would have no way to know their change has no effect.
func warnIfAllowKeylessOverridden(cmd *cobra.Command, cfg *config.Config) {
	if cfg == nil || cfg.AllowKeyless == nil {
		return
	}

	fmt.Fprintf(cmd.ErrOrStderr(),
		"Warning: allow-keyless is set to %t by server config and takes precedence over this change. "+
			"The database was updated, but it will have no effect until the config override is removed.\n",
		*cfg.AllowKeyless)
}

// warnIfAnonAccessOverridden warns on the command's stderr if a server
// config override is masking the anon-access value that was just written to
// the database. Without this, an admin changing the setting via this
// command would have no way to know their change has no effect.
func warnIfAnonAccessOverridden(cmd *cobra.Command, cfg *config.Config) {
	if cfg == nil || cfg.AnonAccess == nil {
		return
	}

	fmt.Fprintf(cmd.ErrOrStderr(),
		"Warning: anon-access is set to %q by server config and takes precedence over this change. "+
			"The database was updated, but it will have no effect until the config override is removed.\n",
		cfg.AnonAccess.String())
}
