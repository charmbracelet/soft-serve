package cmd

import (
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/lexers"
	gansi "github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/server/ui/common"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
)

var (
	lineDigitStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("239"))
	lineBarStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("236"))
	dirnameStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#00AAFF"))
	filenameStyle  = lipgloss.NewStyle()
	filemodeStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#777777"))
)

// blobCommand returns a command that prints the contents of a file.
func blobCommand() *cobra.Command {
	var linenumber bool
	var color bool
	var raw bool

	cmd := &cobra.Command{
		Use:               "blob REPOSITORY [REFERENCE] [PATH]",
		Aliases:           []string{"cat", "show"},
		Short:             "Print out the contents of file at path",
		Args:              cobra.RangeArgs(1, 3),
		PersistentPreRunE: checkIfReadable,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			_, be, _ := fromContext(cmd)
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
				ref = head.Hash.String()
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
					c, err = withFormatting(fp, c)
					if err != nil {
						return err
					}
				}

				if linenumber {
					c = withLineNumber(c, color)
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

func withLineNumber(s string, color bool) string {
	lines := strings.Split(s, "\n")
	// NB: len() is not a particularly safe way to count string width (because
	// it's counting bytes instead of runes) but in this case it's okay
	// because we're only dealing with digits, which are one byte each.
	mll := len(fmt.Sprintf("%d", len(lines)))
	for i, l := range lines {
		digit := fmt.Sprintf("%*d", mll, i+1)
		bar := "│"
		if color {
			digit = lineDigitStyle.Render(digit)
			bar = lineBarStyle.Render(bar)
		}
		if i < len(lines)-1 || len(l) != 0 {
			// If the final line was a newline we'll get an empty string for
			// the final line, so drop the newline altogether.
			lines[i] = fmt.Sprintf(" %s %s %s", digit, bar, l)
		}
	}
	return strings.Join(lines, "\n")
}

func withFormatting(p, c string) (string, error) {
	zero := uint(0)
	lang := ""
	lexer := lexers.Match(p)
	if lexer != nil && lexer.Config() != nil {
		lang = lexer.Config().Name
	}
	formatter := &gansi.CodeBlockElement{
		Code:     c,
		Language: lang,
	}
	r := strings.Builder{}
	styles := common.StyleConfig()
	styles.CodeBlock.Margin = &zero
	rctx := gansi.NewRenderContext(gansi.Options{
		Styles:       styles,
		ColorProfile: termenv.TrueColor,
	})
	err := formatter.Render(&r, rctx)
	if err != nil {
		return "", err
	}
	return r.String(), nil
}
