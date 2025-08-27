// Package code provides code syntax highlighting components.
package code

import (
	"math"
	"strings"
	"sync"

	"github.com/alecthomas/chroma/v2/lexers"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/glamour/v2"
	gansi "github.com/charmbracelet/glamour/v2/ansi"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/soft-serve/pkg/ui/common"
	vp "github.com/charmbracelet/soft-serve/pkg/ui/components/viewport"
)

const (
	defaultTabWidth        = 4
	defaultSideNotePercent = 0.3
)

// Code is a code snippet.
type Code struct {
	*vp.Viewport
	common        common.Common
	sidenote      string
	content       string
	extension     string
	renderContext gansi.RenderContext
	renderMutex   sync.Mutex
	styleConfig   gansi.StyleConfig

	SideNotePercent float64
	TabWidth        int
	ShowLineNumber  bool
	NoContentStyle  lipgloss.Style
	UseGlamour      bool
}

// New returns a new Code.
func New(c common.Common, content, extension string) *Code {
	r := &Code{
		common:          c,
		content:         content,
		extension:       extension,
		TabWidth:        defaultTabWidth,
		SideNotePercent: defaultSideNotePercent,
		Viewport:        vp.New(c),
		NoContentStyle:  c.Styles.NoContent.SetString("No Content."),
	}
	st := common.StyleConfig()
	r.styleConfig = st
	r.renderContext = common.StyleRendererWithStyles(st)
	r.SetSize(c.Width, c.Height)
	return r
}

// SetSize implements common.Component.
func (r *Code) SetSize(width, height int) {
	r.common.SetSize(width, height)
	r.Viewport.SetSize(width, height)
}

// SetContent sets the content of the Code.
func (r *Code) SetContent(c, ext string) tea.Cmd {
	r.content = c
	r.extension = ext
	return r.Init()
}

// SetSideNote sets the sidenote of the Code.
func (r *Code) SetSideNote(s string) tea.Cmd {
	r.sidenote = s
	return r.Init()
}

// Init implements tea.Model.
func (r *Code) Init() tea.Cmd {
	// XXX: We probably won't need the GetHorizontalFrameSize margin
	// subtraction if we get the new viewport soft wrapping to play nicely with
	// Glamour. This also introduces a bug where when it soft wraps, the
	// viewport scrolls left/right for 2 columns on each side of the screen.
	w := r.common.Width - r.common.Styles.App.GetHorizontalFrameSize()
	content := r.content
	if content == "" {
		r.Model.SetContent(r.NoContentStyle.String())
		return nil
	}

	// FIXME chroma & glamour might break wrapping when using tabs since tab
	// width depends on the terminal. This is a workaround to replace tabs with
	// 4-spaces.
	content = strings.ReplaceAll(content, "\t", strings.Repeat(" ", r.TabWidth))

	if r.UseGlamour && common.IsFileMarkdown(content, r.extension) {
		md, err := r.glamourize(w, content)
		if err != nil {
			return common.ErrorCmd(err)
		}
		content = md
	} else {
		f, err := r.renderFile(r.extension, content)
		if err != nil {
			return common.ErrorCmd(err)
		}
		content = f
		if r.ShowLineNumber {
			var ml int
			content, ml = common.FormatLineNumber(r.common.Styles, content, true)
			w -= ml
		}
	}

	if r.sidenote != "" {
		lines := strings.Split(r.sidenote, "\n")
		sideNoteWidth := int(math.Ceil(float64(r.Width()) * r.SideNotePercent))
		for i, l := range lines {
			lines[i] = common.TruncateString(l, sideNoteWidth)
		}
		content = lipgloss.JoinHorizontal(lipgloss.Top, strings.Join(lines, "\n"), content)
	}

	// Fix styles after hard wrapping
	// https://github.com/muesli/reflow/issues/43
	//
	// TODO: solve this upstream in Glamour/Reflow.
	content = lipgloss.NewStyle().Width(w).Render(content)

	r.Model.SetContent(content)

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
	v, cmd := r.Viewport.Update(msg)
	r.Viewport = v.(*vp.Viewport)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	return r, tea.Batch(cmds...)
}

// View implements tea.View.
func (r *Code) View() string {
	return r.Viewport.View()
}

// GotoTop moves the viewport to the top of the log.
func (r *Code) GotoTop() {
	r.Viewport.GotoTop()
}

// GotoBottom moves the viewport to the bottom of the log.
func (r *Code) GotoBottom() {
	r.Viewport.GotoBottom()
}

// HalfViewDown moves the viewport down by half the viewport height.
func (r *Code) HalfViewDown() {
	r.Viewport.HalfViewDown()
}

// HalfViewUp moves the viewport up by half the viewport height.
func (r *Code) HalfViewUp() {
	r.Viewport.HalfViewUp()
}

// ScrollPercent returns the viewport's scroll percentage.
func (r *Code) ScrollPercent() float64 {
	return r.Viewport.ScrollPercent()
}

// ScrollPosition returns the viewport's scroll position.
func (r *Code) ScrollPosition() int {
	scroll := r.ScrollPercent() * 100
	if scroll < 0 || math.IsNaN(scroll) {
		scroll = 0
	}
	return int(scroll)
}

func (r *Code) glamourize(w int, md string) (string, error) {
	r.renderMutex.Lock()
	defer r.renderMutex.Unlock()
	if w > 120 {
		w = 120
	}
	tr, err := glamour.NewTermRenderer(
		glamour.WithStyles(r.styleConfig),
		glamour.WithWordWrap(w),
	)
	if err != nil {
		return "", err //nolint:wrapcheck
	}
	mdt, err := tr.Render(md)
	if err != nil {
		return "", err //nolint:wrapcheck
	}
	return mdt, nil
}

func (r *Code) renderFile(path, content string) (string, error) {
	lexer := lexers.Match(path)
	if path == "" {
		lexer = lexers.Analyse(content)
	}
	lang := ""
	if lexer != nil && lexer.Config() != nil {
		lang = lexer.Config().Name
	}

	formatter := &gansi.CodeBlockElement{
		Code:     content,
		Language: lang,
	}
	s := strings.Builder{}
	rc := r.renderContext
	if r.ShowLineNumber {
		st := common.StyleConfig()
		var m uint
		st.CodeBlock.Margin = &m
		rc = gansi.NewRenderContext(gansi.Options{
			Styles: st,
		})
	}
	err := formatter.Render(&s, rc)
	if err != nil {
		return "", err //nolint:wrapcheck
	}

	return s.String(), nil
}
