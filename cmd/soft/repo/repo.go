package repo

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/soft-serve/cmd"
	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/spf13/cobra"
)

var (
	// Command returns a command for managing repositories.
	Command = &cobra.Command{
		Use:                "repo",
		Aliases:            []string{"repos", "repository", "repositories"},
		Short:              "Manage repositories",
		PersistentPreRunE:  cmd.InitBackendContext,
		PersistentPostRunE: cmd.CloseDBContext,
	}
)

func init() {
	Command.AddCommand(
		blobCommand(),
		branchCommand(),
		collabCommand(),
		commitCommand(),
		createCommand(),
		deleteCommand(),
		descriptionCommand(),
		hiddenCommand(),
		importCommand(),
		infoCommand(),
		listCommand(),
		mirrorCommand(),
		privateCommand(),
		projectName(),
		renameCommand(),
		tagCommand(),
		treeCommand(),
		webhookCommand(),
	)
}

func infoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info REPOSITORY",
		Short: "Get information about a repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			ctx := c.Context()
			be := backend.FromContext(ctx)
			rn := args[0]
			rr, err := be.Repository(ctx, rn)
			if err != nil {
				return err
			}

			if !cmd.CheckUserHasAccess(c, rr.Name(), access.ReadOnlyAccess) {
				return proto.ErrUnauthorized
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
			c.Println(strings.TrimSpace(fmt.Sprint("Project Name: ", rr.ProjectName())))
			c.Println("Repository:", rr.Name())
			c.Println(strings.TrimSpace(fmt.Sprint("Description: ", rr.Description())))
			c.Println("Private:", rr.IsPrivate())
			c.Println("Hidden:", rr.IsHidden())
			c.Println("Mirror:", rr.IsMirror())
			if owner != nil {
				c.Println(strings.TrimSpace(fmt.Sprint("Owner: ", owner.Username())))
			}
			c.Println("Default Branch:", head.Name().Short())
			if len(branches) > 0 {
				c.Println("Branches:")
				for _, b := range branches {
					c.Println("  -", b)
				}
			}
			if len(tags) > 0 {
				c.Println("Tags:")
				for _, t := range tags {
					c.Println("  -", t)
				}
			}

			return nil
		},
	}

	return cmd
}
