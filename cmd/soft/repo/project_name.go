package repo

import (
	"strings"

	"github.com/charmbracelet/soft-serve/cmd"
	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/spf13/cobra"
)

func projectName() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "project-name REPOSITORY [NAME]",
		Aliases: []string{"project"},
		Short:   "Set or get the project name for a repository",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(co *cobra.Command, args []string) error {
			ctx := co.Context()
			be := backend.FromContext(ctx)
			rn := strings.TrimSuffix(args[0], ".git")
			switch len(args) {
			case 1:
				if !cmd.CheckUserHasAccess(co, rn, access.ReadOnlyAccess) {
					return proto.ErrUnauthorized
				}

				pn, err := be.ProjectName(ctx, rn)
				if err != nil {
					return err
				}

				co.Println(pn)
			default:
				if !cmd.CheckUserHasAccess(co, rn, access.ReadWriteAccess) {
					return proto.ErrUnauthorized
				}

				if err := be.SetProjectName(ctx, rn, strings.Join(args[1:], " ")); err != nil {
					return err
				}
			}

			return nil
		},
	}

	return cmd
}
