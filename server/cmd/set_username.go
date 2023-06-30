package cmd

import (
	"github.com/charmbracelet/soft-serve/server/auth"
	"github.com/spf13/cobra"
)

func setUsernameCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-username USERNAME",
		Short: "Set your username",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be, s := fromContext(cmd)
			_, err := be.Authenticate(ctx, auth.NewPublicKey(s.PublicKey()))
			if err != nil {
				return err
			}

			return nil
			// return be.SetUsername(ctx, user.Username(), args[0])
		},
	}

	return cmd
}
