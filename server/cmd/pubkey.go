package cmd

import (
	"github.com/spf13/cobra"
)

func pubkeyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "pubkey",
		Aliases: []string{"pubkeys", "publickey", "publickeys"},
		Short:   "Manage your public keys",
	}
	//
	// pubkeyAddCommand := &cobra.Command{
	// 	Use:   "add AUTHORIZED_KEY",
	// 	Short: "Add a public key",
	// 	Args:  cobra.MinimumNArgs(1),
	// 	RunE: func(cmd *cobra.Command, args []string) error {
	// 		be, s := fromContext(cmd)
	// 		user, err := be.UserByPublicKey(s.PublicKey())
	// 		if err != nil {
	// 			return err
	// 		}
	//
	// 		pk, _, err := backend.ParseAuthorizedKey(strings.Join(args, " "))
	// 		if err != nil {
	// 			return err
	// 		}
	//
	// 		return be.AddPublicKey(user.Username(), pk)
	// 	},
	// }
	//
	// pubkeyRemoveCommand := &cobra.Command{
	// 	Use:   "remove AUTHORIZED_KEY",
	// 	Args:  cobra.MinimumNArgs(1),
	// 	Short: "Remove a public key",
	// 	RunE: func(cmd *cobra.Command, args []string) error {
	// 		be, s := fromContext(cmd)
	// 		user, err := be.UserByPublicKey(s.PublicKey())
	// 		if err != nil {
	// 			return err
	// 		}
	//
	// 		pk, _, err := backend.ParseAuthorizedKey(strings.Join(args, " "))
	// 		if err != nil {
	// 			return err
	// 		}
	//
	// 		return be.RemovePublicKey(user.Username(), pk)
	// 	},
	// }
	//
	// pubkeyListCommand := &cobra.Command{
	// 	Use:     "list",
	// 	Aliases: []string{"ls"},
	// 	Short:   "List public keys",
	// 	Args:    cobra.NoArgs,
	// 	RunE: func(cmd *cobra.Command, args []string) error {
	// 		be, s := fromContext(cmd)
	// 		user, err := be.UserByPublicKey(s.PublicKey())
	// 		if err != nil {
	// 			return err
	// 		}
	//
	// 		pks := user.PublicKeys()
	// 		for _, pk := range pks {
	// 			cmd.Println(backend.MarshalAuthorizedKey(pk))
	// 		}
	//
	// 		return nil
	// 	},
	// }
	//
	// cmd.AddCommand(
	// 	pubkeyAddCommand,
	// 	pubkeyRemoveCommand,
	// 	pubkeyListCommand,
	// )

	return cmd
}
