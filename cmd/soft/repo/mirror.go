package repo

import (
	"github.com/charmbracelet/soft-serve/cmd"
	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/spf13/cobra"
)

func mirrorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "is-mirror REPOSITORY",
		Short: "Whether a repository is a mirror",
		Args:  cobra.ExactArgs(1),
		RunE: func(co *cobra.Command, args []string) error {
			ctx := co.Context()
			be := backend.FromContext(ctx)
			rn := args[0]
			rr, err := be.Repository(ctx, rn)
			if err != nil {
				return err
			}

			if !cmd.CheckUserHasAccess(co, rr.Name(), access.ReadOnlyAccess) {
				return proto.ErrUnauthorized
			}

			isMirror := rr.IsMirror()
			co.Println(isMirror)
			return nil
		},
	}

	return cmd
}
