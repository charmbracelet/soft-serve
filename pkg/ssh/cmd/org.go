package cmd

import (
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/spf13/cobra"
)

// OrgCommand returns a command for managing organizations.
func OrgCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "org",
		Aliases: []string{"orgs", "organization", "organizations"},
		Short:   "Manage organizations",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "create NAME EMAIL",
		Short: "Create a new organization",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			owner := proto.UserFromContext(ctx)
			if owner == nil {
				return proto.ErrUnauthorized
			}
			_, err := be.CreateOrg(ctx, owner, args[0], args[1])
			return err
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List organizations",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			user := proto.UserFromContext(ctx)
			if user == nil {
				return proto.ErrUnauthorized
			}
			orgs, err := be.ListOrgs(ctx, user)
			if err != nil {
				return err
			}
			for _, o := range orgs {
				cmd.Println(o.Name())
			}
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "delete NAME",
		Short: "Delete organization",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			user := proto.UserFromContext(ctx)
			if user == nil {
				return proto.ErrUnauthorized
			}
			return be.DeleteOrganization(ctx, user, args[0])
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "get NAME",
		Short: "Show organization",
		Args:  cobra.ExactArgs(1),
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
			cmd.Println(org.Name())
			return nil
		},
	})

	return cmd
}
