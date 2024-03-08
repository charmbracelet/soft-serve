package cmd

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/ui/common"
	"github.com/charmbracelet/soft-serve/pkg/ui/styles"
	"github.com/spf13/cobra"
)

// blobCommand returns a command that prints the contents of a file.
func blobCommand(renderer *lipgloss.Renderer) *cobra.Command {
	var linenumber bool
	var color bool
	var raw bool

	styles := styles.DefaultStyles(renderer)
	cmd := &cobra.Command{
		Use:               "blob REPOSITORY [REFERENCE] [PATH]",
		Aliases:           []string{"cat", "show"},
		Short:             "Print out the contents of file at path",
		Args:              cobra.RangeArgs(1, 3),
		PersistentPreRunE: checkIfReadable,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			rn := args[0]
			ref := ""
			fp := ""
			switch len(args) {
			case 2:
				fp = args[1]
			case 3:
				ref = args[1]
				fp = args[2]
			}

			repo, err := be.Repository(ctx, rn)
			if err != nil {
				return err
			}

			r, err := repo.Open()
			if err != nil {
				return err
			}

			if ref == "" {
				head, err := r.HEAD()
				if err != nil {
					return err
				}
				ref = head.ID
			}

			tree, err := r.LsTree(ref)
			if err != nil {
				return err
			}

			te, err := tree.TreeEntry(fp)
			if err != nil {
				return err
			}

			if te.Type() != "blob" {
				return git.ErrFileNotFound
			}

			bts, err := te.Contents()
			if err != nil {
				return err
			}

			c := string(bts)
			isBin, _ := te.File().IsBinary()
			if isBin {
				if raw {
					cmd.Println(c)
				} else {
					return fmt.Errorf("binary file: use --raw to print")
				}
			} else {
				if color {
					c, err = common.FormatHighlight(fp, c)
					if err != nil {
						return err
					}
				}

				if linenumber {
					c, _ = common.FormatLineNumber(styles, c, color)
				}

				cmd.Println(c)
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&raw, "raw", "r", false, "Print raw contents")
	cmd.Flags().BoolVarP(&linenumber, "linenumber", "l", false, "Print line numbers")
	cmd.Flags().BoolVarP(&color, "color", "c", false, "Colorize output")

	return cmd
}
