package cmd

import "github.com/spf13/cobra"

func hiddenCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hidden REPOSITORY [TRUE|FALSE]",
		Short: "Hide or unhide a repository",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := fromContext(cmd)
			repo := args[0]
			switch len(args) {
			case 1:
				if err := checkIfReadable(cmd, args); err != nil {
					return err
				}

				hidden, err := cfg.Backend.IsHidden(repo)
				if err != nil {
					return err
				}

				cmd.Println(hidden)
			case 2:
				if err := checkIfCollab(cmd, args); err != nil {
					return err
				}

				hidden := args[1] == "true"
				if err := cfg.Backend.SetHidden(repo, hidden); err != nil {
					return err
				}
			}

			return nil
		},
	}

	return cmd
}
