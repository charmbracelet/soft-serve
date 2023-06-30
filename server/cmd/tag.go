package cmd

import (
	"strings"

	"github.com/spf13/cobra"
)

func tagCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tag",
		Short: "Manage repository tags",
	}

	cmd.AddCommand(
		tagListCommand(),
		tagDeleteCommand(),
	)

	return cmd
}

func tagListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "list REPOSITORY",
		Aliases:           []string{"ls"},
		Short:             "List repository tags",
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: checkIfReadable,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be, _ := fromContext(cmd)
			rn := strings.TrimSuffix(args[0], ".git")
			rr, err := be.Repository(ctx, rn)
			if err != nil {
				return err
			}

			r, err := rr.Open()
			if err != nil {
				return err
			}

			tags, _ := r.Tags()
			for _, t := range tags {
				cmd.Println(t)
			}

			return nil
		},
	}

	return cmd
}

func tagDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "delete REPOSITORY TAG",
		Aliases:           []string{"remove", "rm", "del"},
		Short:             "Delete a tag",
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: checkIfCollab,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be, _ := fromContext(cmd)
			rn := strings.TrimSuffix(args[0], ".git")
			rr, err := be.Repository(ctx, rn)
			if err != nil {
				return err
			}

			r, err := rr.Open()
			if err != nil {
				return err
			}

			return r.DeleteTag(args[1])
		},
	}

	return cmd
}
