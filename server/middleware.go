package server

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/alecthomas/chroma/lexers"
	gansi "github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/lipgloss"
	appCfg "github.com/charmbracelet/soft-serve/internal/config"
	"github.com/charmbracelet/soft-serve/internal/tui/bubbles/git/types"
	"github.com/charmbracelet/wish"
	gitwish "github.com/charmbracelet/wish/git"
	"github.com/gliderlabs/ssh"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/muesli/termenv"
)

var (
	lineDigitStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("239"))
	lineBarStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("236"))
	dirnameStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#00AAFF"))
	filenameStyle  = lipgloss.NewStyle()
	filemodeStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#777777"))
)

type entries []object.TreeEntry

func (cl entries) Len() int      { return len(cl) }
func (cl entries) Swap(i, j int) { cl[i], cl[j] = cl[j], cl[i] }
func (cl entries) Less(i, j int) bool {
	if cl[i].Mode == filemode.Dir && cl[j].Mode == filemode.Dir {
		return cl[i].Name < cl[j].Name
	} else if cl[i].Mode == filemode.Dir {
		return true
	} else if cl[j].Mode == filemode.Dir {
		return false
	} else {
		return cl[i].Name < cl[j].Name
	}
}

// softServeMiddleware is a middleware that handles displaying files with the
// option of syntax highlighting and line numbers.
func softServeMiddleware(ac *appCfg.Config) wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			_, _, active := s.Pty()
			cmds := s.Command()
			if !active && len(cmds) > 0 {
				func() {
					color := false
					lineno := false
					fp := filepath.Clean(cmds[0])
					ps := strings.Split(fp, "/")
					repo := ps[0]
					if repo == "config" {
						return
					}
					repoExists := false
					for _, rp := range ac.Source.AllRepos() {
						if rp.Name() == repo {
							repoExists = true
						}
					}
					if !repoExists {
						s.Write([]byte("repository not found"))
						s.Exit(1)
						return
					}
					auth := ac.AuthRepo(repo, s.PublicKey())
					if auth < gitwish.ReadOnlyAccess {
						s.Write([]byte("unauthorized"))
						s.Exit(1)
						return
					}
					for _, op := range cmds[1:] {
						if op == "-c" || op == "--color" {
							color = true
						} else if op == "-l" || op == "--lineno" || op == "--linenumber" {
							lineno = true
						}
					}
					rs, err := ac.Source.GetRepo(repo)
					if err != nil {
						_, _ = s.Write([]byte(err.Error()))
						_ = s.Exit(1)
						return
					}
					p := strings.Join(ps[1:], "/")
					t, err := rs.LatestTree(p)
					if err != nil && err != object.ErrDirectoryNotFound {
						_, _ = s.Write([]byte(err.Error()))
						_ = s.Exit(1)
						return
					}
					if err == object.ErrDirectoryNotFound {
						fc, err := rs.LatestFile(p)
						if err != nil {
							_, _ = s.Write([]byte(err.Error()))
							_ = s.Exit(1)
							return
						}
						if color {
							ffc, err := withFormatting(fp, fc)
							if err != nil {
								s.Write([]byte(err.Error()))
								s.Exit(1)
								return
							}
							fc = ffc
						}
						if lineno {
							fc = withLineNumber(fc, color)
						}
						s.Write([]byte(fc))
					} else {
						ents := entries(t.Entries)
						sort.Sort(ents)
						for _, e := range ents {
							m, _ := e.Mode.ToOSFileMode()
							if m == 0 {
								s.Write([]byte(strings.Repeat(" ", 10)))
							} else {
								s.Write([]byte(filemodeStyle.Render(m.String())))
							}
							s.Write([]byte(" "))
							if e.Mode.IsFile() {
								s.Write([]byte(filenameStyle.Render(e.Name)))
							} else {
								s.Write([]byte(dirnameStyle.Render(e.Name)))
							}
							s.Write([]byte("\n"))
						}
					}
				}()
			}
			sh(s)
		}
	}
}

func withLineNumber(s string, color bool) string {
	lines := strings.Split(s, "\n")
	// NB: len() is not a particularly safe way to count string width (because
	// it's counting bytes instead of runes) but in this case it's okay
	// because we're only dealing with digits, which are one byte each.
	mll := len(fmt.Sprintf("%d", len(lines)))
	for i, l := range lines {
		digit := fmt.Sprintf("%*d", mll, i)
		line := "│"
		if color {
			digit = lineDigitStyle.Render(digit)
			line = lineBarStyle.Render(line)
		}
		lines[i] = fmt.Sprintf(" %s %s %s", digit, line, l)
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
	styles := types.DefaultStyles()
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
