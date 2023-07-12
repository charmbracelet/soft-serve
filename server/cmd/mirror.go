package cmd

import (
	"github.com/spf13/cobra"
)

func mirrorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "is-mirror REPOSITORY",
		Short:             "Whether a repository is a mirror",
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: checkIfReadable,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			_, be, _ := fromContext(cmd)
			rn := args[0]
			rr, err := be.Repository(ctx, rn)
			if err != nil {
				return err
			}

			isMirror := rr.IsMirror()
			cmd.Println(isMirror)
			return nil
		},
	}

	return cmd
}
