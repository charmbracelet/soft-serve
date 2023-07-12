package cmd

import "github.com/spf13/cobra"

func setUsernameCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-username USERNAME",
		Short: "Set your username",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			_, be, s := fromContext(cmd)
			user, err := be.UserByPublicKey(ctx, s.PublicKey())
			if err != nil {
				return err
			}

			return be.SetUsername(ctx, user.Username(), args[0])
		},
	}

	return cmd
}
