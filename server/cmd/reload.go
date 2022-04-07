package cmd

import (
	gitwish "github.com/charmbracelet/wish/git"
	"github.com/spf13/cobra"
)

// ReloadCommand returns a command that reloads the server configuration.
func ReloadCommand() *cobra.Command {
	reloadCmd := &cobra.Command{
		Use:   "reload",
		Short: "Reloads the configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, s := fromContext(cmd)
			auth := ac.AuthRepo("config", s.PublicKey())
			if auth < gitwish.AdminAccess {
				return ErrUnauthorized
			}
			return ac.Reload()
		},
	}
	return reloadCmd
}
