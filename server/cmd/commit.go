package cmd

import (
	"fmt"
	"strings"

	gansi "github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/server/ui/common"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
)

// commitCommand returns a command that prints the contents of a commit.
func commitCommand() *cobra.Command {
	var color bool

	cmd := &cobra.Command{
		Use:               "commit SHA",
		Short:             "Print out the contents of a diff",
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: checkIfReadable,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := fromContext(cmd)
			repoName := args[0]
			commitSHA := args[1]

			rr, err := cfg.Backend.Repository(repoName)
			if err != nil {
				return err
			}

			r, err := rr.Open()
			if err != nil {
				return err
			}

			raw_commit, err := r.CommitByRevision(commitSHA)

			commit := &git.Commit{
				Commit: raw_commit,
				Hash:   git.Hash(rn),
			}

			patch, err := r.Patch(commit)
			if err != nil {
				return err
			}

			var s strings.Builder
			var pr strings.Builder

			diffChroma := &gansi.CodeBlockElement{
				Code:     patch,
				Language: "diff",
			}

			err = diffChroma.Render(&pr, renderCtx())

			if err != nil {
				s.WriteString(fmt.Sprintf("\n%s", err.Error()))
			} else {
				s.WriteString(fmt.Sprintf("\n%s", pr.String()))
			}

			cmd.Println(s.String())

			return nil
		},
	}

	cmd.Flags().BoolVarP(&color, "color", "c", false, "Colorize output")

	return cmd
}

func renderCtx() gansi.RenderContext {
	return gansi.NewRenderContext(gansi.Options{
		ColorProfile: termenv.TrueColor,
		Styles:       common.StyleConfig(),
	})
}
