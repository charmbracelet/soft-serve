package cmd

import "github.com/spf13/cobra"

func setUsernameCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-username USERNAME",
		Short: "Set your username",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, s := fromContext(cmd)
			user, err := cfg.Backend.UserByPublicKey(s.PublicKey())
			if err != nil {
				return err
			}

			return cfg.Backend.SetUsername(user.Username(), args[0])
		},
	}

	return cmd
}
