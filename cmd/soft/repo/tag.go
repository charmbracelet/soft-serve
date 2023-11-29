package repo

import (
	"strings"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/cmd"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/webhook"
	"github.com/spf13/cobra"
)

func tagCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tag",
		Short: "Manage repository tags",
	}

	cmd.AddCommand(
		tagListCommand(),
		tagDeleteCommand(),
	)

	return cmd
}

func tagListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list REPOSITORY",
		Aliases: []string{"ls"},
		Short:   "List repository tags",
		Args:    cobra.ExactArgs(1),
		RunE: func(co *cobra.Command, args []string) error {
			ctx := co.Context()
			be := backend.FromContext(ctx)
			rn := strings.TrimSuffix(args[0], ".git")
			rr, err := be.Repository(ctx, rn)
			if err != nil {
				return err
			}

			if !cmd.CheckUserHasAccess(co, rr.Name(), access.ReadOnlyAccess) {
				return proto.ErrUnauthorized
			}

			r, err := rr.Open()
			if err != nil {
				return err
			}

			tags, _ := r.Tags()
			for _, t := range tags {
				co.Println(t)
			}

			return nil
		},
	}

	return cmd
}

func tagDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete REPOSITORY TAG",
		Aliases: []string{"remove", "rm", "del"},
		Short:   "Delete a tag",
		Args:    cobra.ExactArgs(2),
		RunE: func(co *cobra.Command, args []string) error {
			ctx := co.Context()
			be := backend.FromContext(ctx)
			rn := strings.TrimSuffix(args[0], ".git")
			rr, err := be.Repository(ctx, rn)
			if err != nil {
				return err
			}

			if !cmd.CheckUserHasAccess(co, rr.Name(), access.ReadWriteAccess) {
				return proto.ErrUnauthorized
			}

			r, err := rr.Open()
			if err != nil {
				log.Errorf("failed to open repo: %s", err)
				return err
			}

			tag := args[1]
			tags, _ := r.Tags()
			var exists bool
			for _, t := range tags {
				if tag == t {
					exists = true
					break
				}
			}

			if !exists {
				log.Errorf("failed to get tag: tag %s does not exist", tag)
				return git.ErrReferenceNotExist
			}

			tagCommit, err := r.TagCommit(tag)
			if err != nil {
				log.Errorf("failed to get tag commit: %s", err)
				return err
			}

			if err := r.DeleteTag(tag); err != nil {
				log.Errorf("failed to delete tag: %s", err)
				return err
			}

			wh, err := webhook.NewBranchTagEvent(ctx, proto.UserFromContext(ctx), rr, git.RefsTags+tag, tagCommit.ID.String(), git.ZeroID)
			if err != nil {
				log.Error("failed to create branch_tag webhook", "err", err)
				return err
			}

			return webhook.SendEvent(ctx, wh)
		},
	}

	return cmd
}
