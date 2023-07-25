package cmd

import (
	"strconv"
	"strings"
	"time"

	"github.com/caarlos0/duration"
	"github.com/caarlos0/tablewriter"
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/proto"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
)

func tokenCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "token",
		Aliases: []string{"access-token"},
		Short:   "Manage access tokens",
	}

	var createExpiresIn string
	createCmd := &cobra.Command{
		Use:   "create NAME",
		Short: "Create a new access token",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			name := strings.Join(args, " ")

			user := proto.UserFromContext(ctx)
			if user == nil {
				return proto.ErrUserNotFound
			}

			var expiresAt time.Time
			var expiresIn time.Duration
			if createExpiresIn != "" {
				d, err := duration.Parse(createExpiresIn)
				if err != nil {
					return err
				}

				expiresIn = d
				expiresAt = time.Now().Add(d)
			}

			token, err := be.CreateAccessToken(ctx, user, name, expiresAt)
			if err != nil {
				return err
			}

			notice := "Access token created"
			if expiresIn != 0 {
				notice += " (expires in " + humanize.Time(expiresAt) + ")"
			}

			cmd.PrintErrln(notice)
			cmd.Println(token)

			return nil
		},
	}

	createCmd.Flags().StringVar(&createExpiresIn, "expires-in", "", "Token expiration time (e.g. 1y, 3mo, 2w, 5d4h, 1h30m)")

	listCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List access tokens",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)

			user := proto.UserFromContext(ctx)
			if user == nil {
				return proto.ErrUserNotFound
			}

			tokens, err := be.ListAccessTokens(ctx, user)
			if err != nil {
				return err
			}

			if len(tokens) == 0 {
				cmd.Println("No tokens found")
				return nil
			}

			now := time.Now()
			return tablewriter.Render(
				cmd.OutOrStdout(),
				tokens,
				[]string{"ID", "Name", "Created At", "Expires In"},
				func(t proto.AccessToken) ([]string, error) {
					expiresAt := "-"
					if !t.ExpiresAt.IsZero() {
						if now.After(t.ExpiresAt) {
							expiresAt = "expired"
						} else {
							expiresAt = humanize.Time(t.ExpiresAt)
						}
					}

					return []string{
						strconv.FormatInt(t.ID, 10),
						t.Name,
						humanize.Time(t.CreatedAt),
						expiresAt,
					}, nil
				},
			)
		},
	}

	deleteCmd := &cobra.Command{
		Use:     "delete ID",
		Aliases: []string{"rm", "remove"},
		Short:   "Delete an access token",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)

			user := proto.UserFromContext(ctx)
			if user == nil {
				return proto.ErrUserNotFound
			}

			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return err
			}

			if err := be.DeleteAccessToken(ctx, user, id); err != nil {
				return err
			}

			cmd.PrintErrln("Access token deleted")
			return nil
		},
	}

	cmd.AddCommand(
		createCmd,
		listCmd,
		deleteCmd,
	)

	return cmd
}
