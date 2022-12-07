package cmd

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/alecthomas/chroma/lexers"
	gansi "github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/proto"
	"github.com/charmbracelet/soft-serve/ui/common"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
)

// RepoCommand is the command for managing repositories.
func RepoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "repo COMMAND",
		Aliases: []string{"repository", "repositories"},
		Short:   "Manage repositories.",
	}
	cmd.AddCommand(
		setCommand(),
		createCommand(),
		deleteCommand(),
		listCommand(),
		showCommand(),
	)
	return cmd
}

func setCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set repository properties.",
	}
	cmd.AddCommand(
		setName(),
		setProjectName(),
		setDescription(),
		setPrivate(),
		setDefaultBranch(),
	)
	return cmd
}

// createCommand is the command for creating a new repository.
func createCommand() *cobra.Command {
	var private bool
	var description string
	var projectName string
	cmd := &cobra.Command{
		Use:   "create REPOSITORY",
		Short: "Create a new repository.",
		Args:  cobra.ExactArgs(1),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cfg, s := fromContext(cmd)
			if !cfg.IsAdmin(s.PublicKey()) {
				return ErrUnauthorized
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := fromContext(cmd)
			name := args[0]
			if err := cfg.Create(name, projectName, description, private); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&private, "private", "p", false, "make the repository private")
	cmd.Flags().StringVarP(&description, "description", "d", "", "set the repository description")
	cmd.Flags().StringVarP(&projectName, "project-name", "n", "", "set the project name")
	return cmd
}

func deleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "delete REPOSITORY",
		Short:             "Delete a repository.",
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := fromContext(cmd)
			name := args[0]
			if err := cfg.Delete(name); err != nil {
				return err
			}
			return nil
		},
	}
	return cmd
}

func checkIfReadable(cmd *cobra.Command, args []string) error {
	var repo string
	if len(args) > 0 {
		repo = args[0]
	}
	cfg, s := fromContext(cmd)
	rn := strings.TrimSuffix(repo, ".git")
	auth := cfg.AuthRepo(rn, s.PublicKey())
	if auth < proto.ReadOnlyAccess {
		return ErrUnauthorized
	}
	return nil
}

func checkIfAdmin(cmd *cobra.Command, args []string) error {
	cfg, s := fromContext(cmd)
	if !cfg.IsAdmin(s.PublicKey()) {
		return ErrUnauthorized
	}
	return nil
}

func checkIfCollab(cmd *cobra.Command, args []string) error {
	var repo string
	if len(args) > 0 {
		repo = args[0]
	}
	cfg, s := fromContext(cmd)
	rn := strings.TrimSuffix(repo, ".git")
	auth := cfg.AuthRepo(rn, s.PublicKey())
	if auth < proto.ReadWriteAccess {
		return ErrUnauthorized
	}
	return nil
}

func setName() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "name REPOSITORY NEW_NAME",
		Short:             "Set the name for a repository.",
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := fromContext(cmd)
			oldName := args[0]
			newName := args[1]
			if err := cfg.Rename(oldName, newName); err != nil {
				return err
			}
			return nil
		},
	}
	return cmd
}

func setProjectName() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "project-name REPOSITORY NAME",
		Short:             "Set the project name for a repository.",
		Args:              cobra.MinimumNArgs(2),
		PersistentPreRunE: checkIfCollab,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := fromContext(cmd)
			rn := strings.TrimSuffix(args[0], ".git")
			if err := cfg.SetProjectName(rn, strings.Join(args[1:], " ")); err != nil {
				return err
			}
			return nil
		},
	}
	return cmd
}

func setDescription() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "description REPOSITORY DESCRIPTION",
		Short:             "Set the description for a repository.",
		Args:              cobra.MinimumNArgs(2),
		PersistentPreRunE: checkIfCollab,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := fromContext(cmd)
			rn := strings.TrimSuffix(args[0], ".git")
			if err := cfg.SetDescription(rn, strings.Join(args[1:], " ")); err != nil {
				return err
			}
			return nil
		},
	}
	return cmd
}

func setPrivate() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "private REPOSITORY [true|false]",
		Short:             "Set a repository to private.",
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: checkIfCollab,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := fromContext(cmd)
			rn := strings.TrimSuffix(args[0], ".git")
			isPrivate, err := strconv.ParseBool(args[1])
			if err != nil {
				return err
			}
			if err := cfg.SetPrivate(rn, isPrivate); err != nil {
				return err
			}
			return nil
		},
	}
	return cmd
}

func setDefaultBranch() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "default-branch REPOSITORY BRANCH",
		Short:             "Set the default branch for a repository.",
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := fromContext(cmd)
			rn := strings.TrimSuffix(args[0], ".git")
			if err := cfg.SetDefaultBranch(rn, args[1]); err != nil {
				return err
			}
			return nil
		},
	}
	return cmd
}

// listCommand returns a command that list file or directory at path.
func listCommand() *cobra.Command {
	listCmd := &cobra.Command{
		Use:               "list PATH",
		Aliases:           []string{"ls"},
		Short:             "List file or directory at path.",
		Args:              cobra.RangeArgs(0, 1),
		PersistentPreRunE: checkIfReadable,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, s := fromContext(cmd)
			rn := ""
			path := ""
			ps := []string{}
			if len(args) > 0 {
				path = filepath.Clean(args[0])
				ps = strings.Split(path, "/")
				rn = strings.TrimSuffix(ps[0], ".git")
				auth := cfg.AuthRepo(rn, s.PublicKey())
				if auth < proto.ReadOnlyAccess {
					return ErrUnauthorized
				}
			}
			if path == "" || path == "." || path == "/" {
				repos, err := cfg.ListRepos()
				if err != nil {
					return err
				}
				for _, r := range repos {
					if cfg.AuthRepo(r.Name(), s.PublicKey()) >= proto.ReadOnlyAccess {
						fmt.Fprintln(s, r.Name())
					}
				}
				return nil
			}
			r, err := cfg.Open(rn)
			if err != nil {
				return err
			}
			head, err := r.Repository().HEAD()
			if err != nil {
				if bs, err := r.Repository().Branches(); err != nil && len(bs) == 0 {
					return fmt.Errorf("repository is empty")
				}
				return err
			}
			tree, err := r.Repository().TreePath(head, "")
			if err != nil {
				return err
			}
			subpath := strings.Join(ps[1:], "/")
			ents := git.Entries{}
			te, err := tree.TreeEntry(subpath)
			if err == git.ErrRevisionNotExist {
				return ErrFileNotFound
			}
			if err != nil {
				return err
			}
			if te.Type() == "tree" {
				tree, err = tree.SubTree(subpath)
				if err != nil {
					return err
				}
				ents, err = tree.Entries()
				if err != nil {
					return err
				}
			} else {
				ents = append(ents, te)
			}
			ents.Sort()
			for _, ent := range ents {
				fmt.Fprintf(s, "%s\t%d\t %s\n", ent.Mode(), ent.Size(), ent.Name())
			}
			return nil
		},
	}
	return listCmd
}

var (
	lineDigitStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("239"))
	lineBarStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("236"))
	dirnameStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#00AAFF"))
	filenameStyle  = lipgloss.NewStyle()
	filemodeStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#777777"))
)

// showCommand returns a command that prints the contents of a file.
func showCommand() *cobra.Command {
	var linenumber bool
	var color bool

	showCmd := &cobra.Command{
		Use:               "show PATH",
		Aliases:           []string{"cat"},
		Short:             "Outputs the contents of the file at path.",
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: checkIfReadable,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, s := fromContext(cmd)
			ps := strings.Split(args[0], "/")
			rn := strings.TrimSuffix(ps[0], ".git")
			fp := strings.Join(ps[1:], "/")
			auth := cfg.AuthRepo(rn, s.PublicKey())
			if auth < proto.ReadOnlyAccess {
				return ErrUnauthorized
			}
			var repo proto.Repository
			repoExists := false
			repos, err := cfg.ListRepos()
			if err != nil {
				return err
			}
			for _, rp := range repos {
				if rp.Name() == rn {
					re, err := rp.Open()
					if err != nil {
						continue
					}
					repoExists = true
					repo = re
					break
				}
			}
			if !repoExists {
				return ErrRepoNotFound
			}
			c, _, err := proto.LatestFile(repo, fp)
			if err != nil {
				return err
			}
			if color {
				c, err = withFormatting(fp, c)
				if err != nil {
					return err
				}
			}
			if linenumber {
				c = withLineNumber(c, color)
			}
			fmt.Fprint(s, c)
			return nil
		},
	}
	showCmd.Flags().BoolVarP(&linenumber, "linenumber", "l", false, "Print line numbers")
	showCmd.Flags().BoolVarP(&color, "color", "c", false, "Colorize output")

	return showCmd
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
