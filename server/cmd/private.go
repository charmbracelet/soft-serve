package cmd

import (
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func privateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "private REPOSITORY [true|false]",
		Short: "Set or get a repository private property",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			_, be, _ := fromContext(cmd)
			rn := strings.TrimSuffix(args[0], ".git")

			switch len(args) {
			case 1:
				if err := checkIfReadable(cmd, args); err != nil {
					return err
				}

				isPrivate, err := be.IsPrivate(ctx, rn)
				if err != nil {
					return err
				}

				cmd.Println(isPrivate)
			case 2:
				isPrivate, err := strconv.ParseBool(args[1])
				if err != nil {
					return err
				}
				if err := checkIfCollab(cmd, args); err != nil {
					return err
				}
				if err := be.SetPrivate(ctx, rn, isPrivate); err != nil {
					return err
				}
			}
			return nil
		},
	}

	return cmd
}
