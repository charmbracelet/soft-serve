package main

import (
	"errors"
	"strings"

	"github.com/charmbracelet/soft-serve/server/auth"
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/sshutils"
	"github.com/spf13/cobra"
)

var (
	password  string
	publicKey string
)

var authCmd = &cobra.Command{
	Use:     "auth [username]",
	Aliases: []string{"authenticate"},
	Short:   "Authenticate user",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		be := backend.FromContext(ctx)

		// TODO: add password support
		// if password == "" && publicKey == "" {
		// 	return errors.New("must specify either a password or public key")
		// }

		publicKey = strings.TrimSpace(publicKey)
		if publicKey == "" {
			return errors.New("must specify a public key")
		}

		pk, _, err := sshutils.ParseAuthorizedKey(publicKey)
		if err != nil {
			return err
		}

		user, err := be.Authenticate(ctx, auth.NewPublicKey(pk))
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()
		if ojson {
			return writeJSON(out, user)
		} else {
			cmd.Printf("User %s authenticated successfully\n", user.Username())
		}

		return nil
	},
}

func init() {
	authCmd.Flags().StringVarP(&password, "password", "p", "", "password")
	authCmd.Flags().StringVarP(&publicKey, "public-key", "k", "", "public key")
}
