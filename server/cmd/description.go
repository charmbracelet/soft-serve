package cmd

import (
	"strings"

	"github.com/spf13/cobra"
)

func descriptionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "description REPOSITORY [DESCRIPTION]",
		Aliases: []string{"desc"},
		Short:   "Set or get the description for a repository",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := fromContext(cmd)
			rn := strings.TrimSuffix(args[0], ".git")
			switch len(args) {
			case 1:
				if err := checkIfReadable(cmd, args); err != nil {
					return err
				}

				desc := cfg.Backend.Description(rn)
				cmd.Println(desc)
			default:
				if err := checkIfCollab(cmd, args); err != nil {
					return err
				}
				if err := cfg.Backend.SetDescription(rn, strings.Join(args[1:], " ")); err != nil {
					return err
				}
			}

			return nil
		},
	}

	return cmd
}
