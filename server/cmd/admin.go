package cmd

import (
	"strings"

	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/spf13/cobra"
)

func adminCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "admin",
		Aliases: []string{"admins"},
		Short:   "Manage admins",
	}

	cmd.AddCommand(
		adminAddCommand(),
		adminRemoveCommand(),
		adminListCommand(),
	)

	return cmd
}

func adminAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "add AUTHORIZED_KEY",
		Short:             "Add an admin",
		Args:              cobra.MinimumNArgs(1),
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := fromContext(cmd)
			pk, c, err := backend.ParseAuthorizedKey(strings.Join(args, " "))
			if err != nil {
				return err
			}

			return cfg.Backend.AddAdmin(pk, c)
		},
	}

	return cmd
}

func adminRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "remove AUTHORIZED_KEY",
		Args:              cobra.MinimumNArgs(1),
		Short:             "Remove an admin",
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := fromContext(cmd)
			pk, _, err := backend.ParseAuthorizedKey(strings.Join(args, " "))
			if err != nil {
				return err
			}

			return cfg.Backend.RemoveAdmin(pk)
		},
	}

	return cmd
}

func adminListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "list",
		Args:              cobra.NoArgs,
		Short:             "List admins",
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, _ := fromContext(cmd)
			admins, err := cfg.Backend.Admins()
			if err != nil {
				return err
			}

			for _, admin := range admins {
				cmd.Println(admin)
			}

			return nil
		},
	}

	return cmd
}
