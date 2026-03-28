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
		Long: `Create a new user.

When passing a public key with -k, shell quoting is stripped by OpenSSH
before the command is transmitted, so an ed25519 key such as:

  ssh host user create alice -k 'ssh-ed25519 AAAA... user@host'

arrives on the server as three separate tokens. Soft Serve re-joins them
automatically, so both of the following forms work:

  -k 'ssh-ed25519 AAAA... user@host'   (quoted, local shell)
  -k ssh-ed25519 AAAA... user@host     (unquoted, same effect over SSH)
`,
		Args:              cobra.MinimumNArgs(1),
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			var pubkeys []ssh.PublicKey
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			username := args[0]

			switch {
			case cmd.Flags().Changed("key") && key == "":
				// -k was supplied with an empty value.
				return fmt.Errorf("flag --key requires a non-empty public key")
			case cmd.Flags().Changed("key"):
				// Re-join the -k value with any remaining positional args.
				// This reconstructs a key split across tokens by SSH quoting
				// stripping (e.g. 'ssh-ed25519 AAAA' → two tokens).
				keyStr := strings.TrimSpace(strings.Join(append([]string{key}, args[1:]...), " "))
				pk, _, err := sshutils.ParseAuthorizedKey(keyStr)
				if err != nil {
					return err
				}
				pubkeys = []ssh.PublicKey{pk}
			case len(args) > 1:
				return fmt.Errorf("unexpected arguments: %s", strings.Join(args[1:], " "))
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
