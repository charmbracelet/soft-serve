package cmd

import (
	"github.com/charmbracelet/soft-serve/proto"
	"github.com/spf13/cobra"
)

// ReloadCommand returns a command that reloads the server configuration.
func ReloadCommand() *cobra.Command {
	reloadCmd := &cobra.Command{
		Use:   "reload",
		Short: "Reloads the configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, s := fromContext(cmd)
			auth := cfg.AuthRepo("config", s.PublicKey())
			if auth < proto.AdminAccess {
				return ErrUnauthorized
			}
			return nil
		},
	}
	return reloadCmd
}
