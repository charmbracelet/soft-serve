package cmd

import "github.com/spf13/cobra"

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
		createCommand(),
		deleteCommand(),
		descriptionCommand(),
		importCommand(),
		listCommand(),
		privateCommand(),
		projectName(),
		renameCommand(),
		tagCommand(),
		treeCommand(),
	)

	return cmd
}
