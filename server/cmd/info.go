package cmd

import (
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/spf13/cobra"
)

func infoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show your info",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, s := fromContext(cmd)
			user, err := cfg.Backend.UserByPublicKey(s.PublicKey())
			if err != nil {
				return err
			}

			cmd.Printf("Username: %s\n", user.Username())
			cmd.Printf("Admin: %t\n", user.IsAdmin())
			cmd.Printf("Public keys:\n")
			for _, pk := range user.PublicKeys() {
				cmd.Printf("  %s\n", backend.MarshalAuthorizedKey(pk))
			}
			return nil
		},
	}

	return cmd
}
