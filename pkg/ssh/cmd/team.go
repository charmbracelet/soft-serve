package cmd

import (
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/spf13/cobra"
)

// TeamCommand returns a command for managing teams.
func TeamCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "team",
		Aliases: []string{"teams"},
		Short:   "Manage teams",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "create ORG NAME",
		Short: "Create a new team",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			user := proto.UserFromContext(ctx)
			if user == nil {
				return proto.ErrUnauthorized
			}

			org, err := be.FindOrganization(ctx, user, args[0])
			if err != nil {
				return err
			}

			team, err := be.CreateTeam(ctx, org, user, args[1])
			if err != nil {
				return err
			}

			cmd.Println("Created", team.Name())

			return err
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List teams",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			user := proto.UserFromContext(ctx)
			if user == nil {
				return proto.ErrUnauthorized
			}
			teams, err := be.ListTeams(ctx, user)
			if err != nil {
				return err
			}
			for _, o := range teams {
				cmd.Println(o.Name())
			}
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "delete ORG NAME",
		Short: "Delete team",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			user := proto.UserFromContext(ctx)
			if user == nil {
				return proto.ErrUnauthorized
			}

			org, err := be.FindOrganization(ctx, user, args[0])
			if err != nil {
				return err
			}

			team, err := be.GetTeam(ctx, user, org, args[1])
			if err != nil {
				return err
			}

			return be.DeleteTeam(ctx, user, team)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "get ORG NAME",
		Short: "Show team",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			user := proto.UserFromContext(ctx)
			if user == nil {
				return proto.ErrUnauthorized
			}

			org, err := be.FindOrganization(ctx, user, args[0])
			if err != nil {
				return err
			}

			team, err := be.GetTeam(ctx, user, org, args[1])
			if err != nil {
				return err
			}

			cmd.Println(org.Handle(), "/", team.Name())
			return nil
		},
	})

	return cmd
}
