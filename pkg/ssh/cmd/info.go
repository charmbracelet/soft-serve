package cmd

import (
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/sshutils"
	"github.com/spf13/cobra"
)

// InfoCommand returns a command that shows the user's info
func InfoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show your info",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			pk := sshutils.PublicKeyFromContext(ctx)
			user, err := be.UserByPublicKey(ctx, pk)
			if err != nil {
				return err
			}

			cmd.Printf("Username: %s\n", user.Username())
			cmd.Printf("Admin: %t\n", user.IsAdmin())
			cmd.Printf("Public keys:\n")
			for _, pk := range user.PublicKeys() {
				cmd.Printf("  %s\n", sshutils.MarshalAuthorizedKey(pk))
			}
			return nil
		},
	}

	return cmd
}
