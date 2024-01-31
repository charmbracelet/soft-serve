package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/spf13/cobra"
)

// RepoCommand returns a command for managing repositories.
func RepoCommand(renderer *lipgloss.Renderer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "repo",
		Aliases: []string{"repos", "repository", "repositories"},
		Short:   "Manage repositories",
	}

	cmd.AddCommand(
		blobCommand(renderer),
		branchCommand(),
		collabCommand(),
		commitCommand(renderer),
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
		webhookCommand(),
	)

	cmd.AddCommand(
		&cobra.Command{
			Use:               "info REPOSITORY",
			Short:             "Get information about a repository",
			Args:              cobra.ExactArgs(1),
			PersistentPreRunE: checkIfReadable,
			RunE: func(cmd *cobra.Command, args []string) error {
				ctx := cmd.Context()
				be := backend.FromContext(ctx)
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

				var owner proto.User
				if rr.UserID() > 0 {
					owner, err = be.UserByID(ctx, rr.UserID())
					if err != nil {
						return err
					}
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
				if owner != nil {
					cmd.Println(strings.TrimSpace(fmt.Sprint("Owner: ", owner.Username())))
				}
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
