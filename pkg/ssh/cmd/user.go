package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/sshutils"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

// UserCommand returns the user subcommand.
func UserCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "user",
		Aliases: []string{"users"},
		Short:   "Manage users",
	}

	var admin bool
	var key string
	userCreateCommand := &cobra.Command{
		Use:   "create USERNAME",
		Short: "Create a new user",
		Long: `Create a new user. The public key can be provided via the -k/--key flag.
When connecting over SSH, the key is automatically reconstructed if the
SSH protocol splits it on spaces.`,
		Args:              cobra.MinimumNArgs(1),
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			var pubkeys []ssh.PublicKey
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			username := args[0]
			// SSH exec tokenizes commands on spaces, so a public key like
			// "ssh-ed25519 AAAA..." arrives as separate args. Reconstruct
			// the full key by joining the flag value with any trailing args.
			// Use a local variable to avoid mutating the closure-captured flag
			// var, which would bleed state into subsequent invocations.
			localKey := key
			if cmd.Flags().Changed("key") && localKey == "" {
				return fmt.Errorf("--key flag requires a non-empty public key value")
			}
			if len(args) > 1 {
				// SSH exec tokenizes the command on spaces, splitting "ssh-ed25519 AAAA..."
				// into separate args. Reconstruct only when localKey is a partial key type
				// (no space = just "ssh-ed25519") or when no -k flag was given.
				if strings.Contains(localKey, " ") {
					// localKey is already a complete key; extra positional args are unexpected
					return fmt.Errorf("unexpected arguments: %v", args[1:])
				}
				parts := args[1:]
				if localKey != "" {
					parts = append([]string{localKey}, parts...)
				}
				localKey = strings.Join(parts, " ")
			}
			if localKey != "" {
				pk, _, err := sshutils.ParseAuthorizedKey(localKey)
				if err != nil {
					return err
				}

				pubkeys = []ssh.PublicKey{pk}
			}

			opts := proto.UserOptions{
				Admin:      admin,
				PublicKeys: pubkeys,
			}

			_, err := be.CreateUser(ctx, username, opts)
			return err
		},
	}

	userCreateCommand.Flags().BoolVarP(&admin, "admin", "a", false, "make the user an admin")
	userCreateCommand.Flags().StringVarP(&key, "key", "k", "", "add a public key to the user")

	userDeleteCommand := &cobra.Command{
		Use:               "delete USERNAME",
		Short:             "Delete a user",
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			username := args[0]

			return be.DeleteUser(ctx, username)
		},
	}

	userListCommand := &cobra.Command{
		Use:               "list",
		Aliases:           []string{"ls"},
		Short:             "List users",
		Args:              cobra.NoArgs,
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			users, err := be.Users(ctx)
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
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			username := args[0]
			pubkey := strings.Join(args[1:], " ")
			pk, _, err := sshutils.ParseAuthorizedKey(pubkey)
			if err != nil {
				return err
			}

			return be.AddPublicKey(ctx, username, pk)
		},
	}

	userRemovePubkeyCommand := &cobra.Command{
		Use:               "remove-pubkey USERNAME AUTHORIZED_KEY",
		Short:             "Remove a public key from a user",
		Args:              cobra.MinimumNArgs(2),
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			username := args[0]
			pubkey := strings.Join(args[1:], " ")
			pk, _, err := sshutils.ParseAuthorizedKey(pubkey)
			if err != nil {
				return err
			}

			return be.RemovePublicKey(ctx, username, pk)
		},
	}

	userSetAdminCommand := &cobra.Command{
		Use:               "set-admin USERNAME [true|false]",
		Short:             "Make a user an admin",
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			username := args[0]

			return be.SetAdmin(ctx, username, args[1] == "true")
		},
	}

	userInfoCommand := &cobra.Command{
		Use:               "info USERNAME",
		Short:             "Show information about a user",
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			username := args[0]

			user, err := be.User(ctx, username)
			if err != nil {
				return err
			}

			isAdmin := user.IsAdmin()

			cmd.Printf("Username: %s\n", user.Username())
			cmd.Printf("Admin: %t\n", isAdmin)
			cmd.Printf("Public keys:\n")
			for _, pk := range user.PublicKeys() {
				cmd.Printf("  %s\n", sshutils.MarshalAuthorizedKey(pk))
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
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			username := args[0]
			newUsername := args[1]

			return be.SetUsername(ctx, username, newUsername)
		},
	}

	cmd.AddCommand(
		userCreateCommand,
		userAddPubkeyCommand,
		userInfoCommand,
		userListCommand,
		userDeleteCommand,
		userRemovePubkeyCommand,
		userSetAdminCommand,
		userSetUsernameCommand,
	)

	return cmd
}
