package cmd

import (
	"sort"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

func userCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "user",
		Aliases: []string{"users"},
		Short:   "Manage users",
	}

	var admin bool
	var key string
	userAddCommand := &cobra.Command{
		Use:               "add USERNAME",
		Short:             "Add a user",
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := fromContext(cmd)
			username := args[0]
			pk, _, err := backend.ParseAuthorizedKey(key)
			if err != nil {
				return err
			}

			opts := backend.UserOptions{
				Admin:      admin,
				PublicKeys: []ssh.PublicKey{pk},
			}

			_, err = cfg.Backend.CreateUser(username, opts)
			return err
		},
	}

	userAddCommand.Flags().BoolVarP(&admin, "admin", "a", false, "make the user an admin")
	userAddCommand.Flags().StringVarP(&key, "key", "k", "", "add a public key to the user")

	userRemoveCommand := &cobra.Command{
		Use:               "remove USERNAME",
		Short:             "Remove a user",
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := fromContext(cmd)
			username := args[0]

			return cfg.Backend.DeleteUser(username)
		},
	}

	userListCommand := &cobra.Command{
		Use:               "list",
		Aliases:           []string{"ls"},
		Short:             "List users",
		Args:              cobra.NoArgs,
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, _ := fromContext(cmd)
			users, err := cfg.Backend.Users()
			if err != nil {
				return err
			}

			sort.Strings(users)
			for _, u := range users {
				cmd.Println(u)
			}

			return nil
		},
	}

	userAddPubkeyCommand := &cobra.Command{
		Use:               "add-pubkey USERNAME AUTHORIZED_KEY",
		Short:             "Add a public key to a user",
		Args:              cobra.MinimumNArgs(2),
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := fromContext(cmd)
			username := args[0]
			pubkey := strings.Join(args[1:], " ")
			pk, _, err := backend.ParseAuthorizedKey(pubkey)
			if err != nil {
				return err
			}

			return cfg.Backend.AddPublicKey(username, pk)
		},
	}

	userRemovePubkeyCommand := &cobra.Command{
		Use:               "remove-pubkey USERNAME AUTHORIZED_KEY",
		Short:             "Remove a public key from a user",
		Args:              cobra.MinimumNArgs(2),
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := fromContext(cmd)
			username := args[0]
			pubkey := strings.Join(args[1:], " ")
			log.Debugf("key is %q", pubkey)
			pk, _, err := backend.ParseAuthorizedKey(pubkey)
			if err != nil {
				return err
			}

			return cfg.Backend.RemovePublicKey(username, pk)
		},
	}

	userSetAdminCommand := &cobra.Command{
		Use:               "set-admin USERNAME [true|false]",
		Short:             "Make a user an admin",
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := fromContext(cmd)
			username := args[0]

			return cfg.Backend.SetAdmin(username, args[1] == "true")
		},
	}

	userInfoCommand := &cobra.Command{
		Use:               "info USERNAME",
		Short:             "Show information about a user",
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, s := fromContext(cmd)
			ak := backend.MarshalAuthorizedKey(s.PublicKey())
			username := args[0]

			user, err := cfg.Backend.User(username)
			if err != nil {
				return err
			}

			isAdmin := user.IsAdmin()
			for _, k := range cfg.InitialAdminKeys {
				if ak == k {
					isAdmin = true
					break
				}
			}

			cmd.Printf("Username: %s\n", user.Username())
			cmd.Printf("Admin: %t\n", isAdmin)
			cmd.Printf("Public keys:\n")
			for _, pk := range user.PublicKeys() {
				cmd.Printf("  %s\n", backend.MarshalAuthorizedKey(pk))
			}

			return nil
		},
	}

	userSetUsernameCommand := &cobra.Command{
		Use:               "set-username USERNAME NEW_USERNAME",
		Short:             "Change a user's username",
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := fromContext(cmd)
			username := args[0]
			newUsername := args[1]

			return cfg.Backend.SetUsername(username, newUsername)
		},
	}

	cmd.AddCommand(
		userAddCommand,
		userAddPubkeyCommand,
		userInfoCommand,
		userListCommand,
		userRemoveCommand,
		userRemovePubkeyCommand,
		userSetAdminCommand,
		userSetUsernameCommand,
	)

	return cmd
}
