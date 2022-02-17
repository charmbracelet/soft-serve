package server

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/lexers"
	gansi "github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/lipgloss"
	appCfg "github.com/charmbracelet/soft-serve/internal/config"
	"github.com/charmbracelet/soft-serve/internal/tui/bubbles/git/types"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/git"
	"github.com/gliderlabs/ssh"
	gg "github.com/go-git/go-git/v5"
	"github.com/muesli/termenv"
)

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
						return
					}
					auth := ac.AuthRepo(repo, s.PublicKey())
					if auth < git.ReadOnlyAccess {
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
					fc, err := readFile(rs.Repository(), strings.Join(ps[1:], "/"))
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
				}()
			}
			sh(s)
		}
	}
}

func readFile(r *gg.Repository, fp string) (string, error) {
	l, err := r.Log(&gg.LogOptions{})
	if err != nil {
		return "", err
	}
	c, err := l.Next()
	if err != nil {
		return "", err
	}
	f, err := c.File(fp)
	if err != nil {
		return "", err
	}
	fc, err := f.Contents()
	if err != nil {
		return "", err
	}
	return fc, nil
}

func withLineNumber(s string, color bool) string {
	st := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	lines := strings.Split(s, "\n")
	mll := fmt.Sprintf("%d", len(fmt.Sprintf("%d", len(lines))))
	for i, l := range lines {
		lines[i] = fmt.Sprintf("%-"+mll+"d │ %s", i+1, l)
		if color {
			lines[i] = st.Render(lines[i])
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
