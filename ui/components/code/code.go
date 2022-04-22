package code

import (
	"strings"

	"github.com/alecthomas/chroma/lexers"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	gansi "github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/soft-serve/ui/common"
	vp "github.com/charmbracelet/soft-serve/ui/components/viewport"
	"github.com/muesli/reflow/wrap"
	"github.com/muesli/termenv"
)

// Code is a code snippet.
type Code struct {
	common         common.Common
	content        string
	extension      string
	viewport       *vp.Viewport
	NoContentStyle lipgloss.Style
}

// New returns a new Code.
func New(c common.Common, content, extension string) *Code {
	r := &Code{
		common:         c,
		content:        content,
		extension:      extension,
		viewport:       vp.New(),
		NoContentStyle: c.Styles.CodeNoContent.Copy(),
	}
	r.SetSize(c.Width, c.Height)
	return r
}

// SetSize implements common.Component.
func (r *Code) SetSize(width, height int) {
	r.common.SetSize(width, height)
	r.viewport.SetSize(width, height)
}

// SetContent sets the content of the Code.
func (r *Code) SetContent(c, ext string) tea.Cmd {
	r.content = c
	r.extension = ext
	return r.Init()
}

// GotoTop reset the viewport to the top.
func (r *Code) GotoTop() {
	r.viewport.Viewport.GotoTop()
}

// Init implements tea.Model.
func (r *Code) Init() tea.Cmd {
	w := r.common.Width
	c := r.content
	if c == "" {
		c = r.NoContentStyle.String()
	}
	f, err := renderFile(r.extension, c, w)
	if err != nil {
		return common.ErrorCmd(err)
	}
	// FIXME: this is a hack to reset formatting at the end of every line.
	c = wrap.String(f, w)
	s := strings.Split(c, "\n")
	for i, l := range s {
		s[i] = l + "\x1b[0m"
	}
	r.viewport.Viewport.SetContent(strings.Join(s, "\n"))
	return nil
}

// Update implements tea.Model.
func (r *Code) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg.(type) {
	case tea.WindowSizeMsg:
		// Recalculate content width and line wrap.
		cmds = append(cmds, r.Init())
	}
	v, cmd := r.viewport.Update(msg)
	r.viewport = v.(*vp.Viewport)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	return r, tea.Batch(cmds...)
}

// View implements tea.View.
func (r *Code) View() string {
	return r.viewport.View()
}

func styleConfig() gansi.StyleConfig {
	noColor := ""
	s := glamour.DarkStyleConfig
	s.Document.StylePrimitive.Color = &noColor
	s.CodeBlock.Chroma.Text.Color = &noColor
	s.CodeBlock.Chroma.Name.Color = &noColor
	return s
}

func renderCtx() gansi.RenderContext {
	return gansi.NewRenderContext(gansi.Options{
		ColorProfile: termenv.TrueColor,
		Styles:       styleConfig(),
	})
}

func glamourize(w int, md string) (string, error) {
	if w > 120 {
		w = 120
	}
	tr, err := glamour.NewTermRenderer(
		glamour.WithStyles(styleConfig()),
		glamour.WithWordWrap(w),
	)

	if err != nil {
		return "", err
	}
	mdt, err := tr.Render(md)
	if err != nil {
		return "", err
	}
	return mdt, nil
}

func renderFile(path, content string, width int) (string, error) {
	lexer := lexers.Fallback
	if path == "" {
		lexer = lexers.Analyse(content)
	} else {
		lexer = lexers.Match(path)
	}
	lang := ""
	if lexer != nil && lexer.Config() != nil {
		lang = lexer.Config().Name
	}
	if lang == "markdown" {
		md, err := glamourize(width, content)
		if err != nil {
			return "", err
		}
		return md, nil
	}
	formatter := &gansi.CodeBlockElement{
		Code:     content,
		Language: lang,
	}
	r := strings.Builder{}
	err := formatter.Render(&r, renderCtx())
	if err != nil {
		return "", err
	}
	return r.String(), nil
}
