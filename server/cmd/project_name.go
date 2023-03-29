package cmd

import (
	"strings"

	"github.com/spf13/cobra"
)

func projectName() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "project-name REPOSITORY [NAME]",
		Aliases: []string{"project"},
		Short:   "Set or get the project name for a repository",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := fromContext(cmd)
			rn := strings.TrimSuffix(args[0], ".git")
			switch len(args) {
			case 1:
				if err := checkIfReadable(cmd, args); err != nil {
					return err
				}

				pn := cfg.Backend.ProjectName(rn)
				cmd.Println(pn)
			default:
				if err := checkIfCollab(cmd, args); err != nil {
					return err
				}
				if err := cfg.Backend.SetProjectName(rn, strings.Join(args[1:], " ")); err != nil {
					return err
				}
			}

			return nil
		},
	}

	return cmd
}
