package cmd

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2/table"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
)

func pushMirrorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "push-mirror",
		Aliases: []string{"push-mirrors"},
		Short:   "Manage repository push mirrors",
	}

	cmd.AddCommand(
		pushMirrorAddCommand(),
		pushMirrorRemoveCommand(),
		pushMirrorListCommand(),
	)

	return cmd
}

func pushMirrorAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "add REPOSITORY NAME REMOTE_URL",
		Short:             "Add a push mirror to a repository",
		Args:              cobra.ExactArgs(3),
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)

			remoteURL := args[2]
			if err := validateMirrorURL(remoteURL); err != nil {
				return err
			}

			repo, err := be.Repository(ctx, args[0])
			if err != nil {
				return err
			}
			return be.AddPushMirror(ctx, repo, args[1], remoteURL)
		},
	}
	return cmd
}

func pushMirrorRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "remove REPOSITORY NAME",
		Short:             "Remove a push mirror from a repository",
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repo, err := be.Repository(ctx, args[0])
			if err != nil {
				return err
			}
			return be.RemovePushMirror(ctx, repo, args[1])
		},
	}
	return cmd
}

// validateMirrorURL ensures the remote URL uses an allowed protocol.
// Rejects file://, git://, and other protocols that could be used for SSRF.
func validateMirrorURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid mirror URL: %w", err)
	}
	switch strings.ToLower(u.Scheme) {
	case "https", "http", "ssh":
		return nil
	case "":
		// SCP-style SSH remote like git@github.com:org/repo.git — allowed
		return nil
	default:
		return fmt.Errorf("mirror URL scheme %q is not allowed (use https, http, or ssh)", u.Scheme)
	}
}

func pushMirrorListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "list REPOSITORY",
		Short:             "List push mirrors for a repository",
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: checkIfReadable,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repo, err := be.Repository(ctx, args[0])
			if err != nil {
				return err
			}

			mirrors, err := be.ListPushMirrors(ctx, repo)
			if err != nil {
				return err
			}

			t := table.New().Headers("ID", "Name", "Remote URL", "Enabled", "Created At", "Updated At")
			for _, m := range mirrors {
				t = t.Row(
					strconv.FormatInt(m.ID, 10),
					m.Name,
					m.RemoteURL,
					strconv.FormatBool(m.Enabled),
					humanize.Time(m.CreatedAt),
					humanize.Time(m.UpdatedAt),
				)
			}
			cmd.Println(t)
			return nil
		},
	}
	return cmd
}
