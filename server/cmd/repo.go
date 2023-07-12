package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func repoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "repo",
		Aliases: []string{"repos", "repository", "repositories"},
		Short:   "Manage repositories",
	}

	cmd.AddCommand(
		blobCommand(),
		branchCommand(),
		collabCommand(),
		commitCommand(),
		createCommand(),
		deleteCommand(),
		descriptionCommand(),
		hiddenCommand(),
		importCommand(),
		listCommand(),
		mirrorCommand(),
		privateCommand(),
		projectName(),
		renameCommand(),
		tagCommand(),
		treeCommand(),
	)

	cmd.AddCommand(
		&cobra.Command{
			Use:               "info REPOSITORY",
			Short:             "Get information about a repository",
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

				r, err := rr.Open()
				if err != nil {
					return err
				}

				head, err := r.HEAD()
				if err != nil {
					return err
				}

				branches, _ := r.Branches()
				tags, _ := r.Tags()

				// project name and description are optional, handle trailing
				// whitespace to avoid breaking tests.
				cmd.Println(strings.TrimSpace(fmt.Sprint("Project Name: ", rr.ProjectName())))
				cmd.Println("Repository:", rr.Name())
				cmd.Println(strings.TrimSpace(fmt.Sprint("Description: ", rr.Description())))
				cmd.Println("Private:", rr.IsPrivate())
				cmd.Println("Hidden:", rr.IsHidden())
				cmd.Println("Mirror:", rr.IsMirror())
				cmd.Println("Default Branch:", head.Name().Short())
				if len(branches) > 0 {
					cmd.Println("Branches:")
					for _, b := range branches {
						cmd.Println("  -", b)
					}
				}
				if len(tags) > 0 {
					cmd.Println("Tags:")
					for _, t := range tags {
						cmd.Println("  -", t)
					}
				}

				return nil
			},
		},
	)

	return cmd
}
