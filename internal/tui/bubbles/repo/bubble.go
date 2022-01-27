package repo

import (
	"bytes"
	"fmt"
	"strconv"
	"text/template"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/soft-serve/internal/git"
	"github.com/charmbracelet/soft-serve/internal/tui/style"
	"github.com/muesli/reflow/truncate"
	"github.com/muesli/reflow/wrap"
)

const (
	glamourMaxWidth  = 120
	repoNameMaxWidth = 32
)

var glamourStyle = func() ansi.StyleConfig {
	noColor := ""
	s := glamour.DarkStyleConfig
	s.Document.StylePrimitive.Color = &noColor
	s.CodeBlock.Chroma.Text.Color = &noColor
	s.CodeBlock.Chroma.Name.Color = &noColor
	return s
}()

type ErrMsg struct {
	Error error
}

type Bubble struct {
	templateObject interface{}
	repoSource     *git.RepoSource
	name           string
	repo           *git.Repo
	styles         *style.Styles
	readmeViewport *ViewportBubble
	readme         string
	height         int
	heightMargin   int
	width          int
	widthMargin    int
	Active         bool

	// XXX: ideally, we get these from the parent as a pointer. Currently, we
	// can't add a *tui.Config because it's an illegal import cycle. One
	// solution would be to (rename and) move this Bubble into the parent
	// package.
	Host string
	Port int
}

func NewBubble(rs *git.RepoSource, name string, styles *style.Styles, width, wm, height, hm int, tmp interface{}) *Bubble {
	b := &Bubble{
		templateObject: tmp,
		repoSource:     rs,
		name:           name,
		styles:         styles,
		heightMargin:   hm,
		widthMargin:    wm,
		readmeViewport: &ViewportBubble{
			Viewport: &viewport.Model{},
		},
	}
	b.SetSize(width, height)
	return b
}

func (b *Bubble) Init() tea.Cmd {
	return b.setupCmd
}

func (b *Bubble) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		b.SetSize(msg.Width, msg.Height)
		// XXX: if we find that longer readmes take more than a few
		// milliseconds to render we may need to move Glamour rendering into a
		// command.
		md, err := b.glamourize(b.readme)
		if err != nil {
			return b, nil
		}
		b.readmeViewport.Viewport.SetContent(md)
	}
	rv, cmd := b.readmeViewport.Update(msg)
	b.readmeViewport = rv.(*ViewportBubble)
	cmds = append(cmds, cmd)
	return b, tea.Batch(cmds...)
}

func (b *Bubble) SetSize(w, h int) {
	b.width = w
	b.height = h
	b.readmeViewport.Viewport.Width = w - b.widthMargin
	b.readmeViewport.Viewport.Height = h - lipgloss.Height(b.headerView()) - b.heightMargin
}

func (b *Bubble) GotoTop() {
	b.readmeViewport.Viewport.GotoTop()
}

func (b Bubble) headerView() string {
	// Render repo title
	title := b.name
	if title == "config" {
		title = "Home"
	}
	title = truncate.StringWithTail(title, repoNameMaxWidth, "…")
	title = b.styles.RepoTitle.Render(title)

	// Render clone command
	var note string
	if b.name == "config" {
		note = ""
	} else {
		note = fmt.Sprintf("git clone %s", b.sshAddress())
	}
	noteWidth := b.width -
		b.widthMargin -
		lipgloss.Width(title) -
		b.styles.RepoTitleBox.GetHorizontalFrameSize()
	// Hard-wrap the clone command only, without the usual word-wrapping. since
	// a long repo name isn't going to be a series of space-separated "words",
	// we'll always want it to be perfectly hard-wrapped.
	note = wrap.String(note, noteWidth-b.styles.RepoNote.GetHorizontalFrameSize())
	note = b.styles.RepoNote.Copy().Width(noteWidth).Render(note)

	// Render borders on name and command
	height := max(lipgloss.Height(title), lipgloss.Height(note))
	titleBoxStyle := b.styles.RepoTitleBox.Copy().Height(height)
	noteBoxStyle := b.styles.RepoNoteBox.Copy().Height(height)
	if b.Active {
		titleBoxStyle = titleBoxStyle.BorderForeground(b.styles.ActiveBorderColor)
		noteBoxStyle = noteBoxStyle.BorderForeground(b.styles.ActiveBorderColor)
	}
	title = titleBoxStyle.Render(title)
	note = noteBoxStyle.Render(note)

	// Render
	return lipgloss.JoinHorizontal(lipgloss.Top, title, note)
}

func (b *Bubble) View() string {
	header := b.headerView()
	bs := b.styles.RepoBody.Copy()
	if b.Active {
		bs = bs.BorderForeground(b.styles.ActiveBorderColor)
	}
	body := bs.Width(b.width - b.widthMargin - b.styles.RepoBody.GetVerticalFrameSize()).
		Height(b.height - b.heightMargin - lipgloss.Height(header)).
		Render(b.readmeViewport.View())
	return header + body
}

func (b Bubble) sshAddress() string {
	p := ":" + strconv.Itoa(int(b.Port))
	if p == ":22" {
		p = ""
	}
	return fmt.Sprintf("ssh://%s%s/%s", b.Host, p, b.name)
}

func (b *Bubble) setupCmd() tea.Msg {
	r, err := b.repoSource.GetRepo(b.name)
	if err == git.ErrMissingRepo {
		return nil
	}
	if err != nil {
		return ErrMsg{err}
	}
	md := r.Readme
	if b.templateObject != nil {
		md, err = b.templatize(md)
		if err != nil {
			return ErrMsg{err}
		}
	}
	b.readme = md
	md, err = b.glamourize(md)
	if err != nil {
		return ErrMsg{err}
	}
	b.readmeViewport.Viewport.SetContent(md)
	b.GotoTop()
	return nil
}

func (b *Bubble) templatize(mdt string) (string, error) {
	t, err := template.New("readme").Parse(mdt)
	if err != nil {
		return "", err
	}
	buf := &bytes.Buffer{}
	err = t.Execute(buf, b.templateObject)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (b *Bubble) glamourize(md string) (string, error) {
	w := b.width - b.widthMargin - b.styles.RepoBody.GetHorizontalFrameSize()
	if w > glamourMaxWidth {
		w = glamourMaxWidth
	}
	tr, err := glamour.NewTermRenderer(
		glamour.WithStyles(glamourStyle),
		glamour.WithWordWrap(w),
	)

	if err != nil {
		return "", err
	}
	mdt, err := tr.Render(md)
	if err != nil {
		return "", err
	}
	// For now, hard-wrap long lines in Glamour that would otherwise break the
	// layout when wrapping. This may be due to #43 in Reflow, which has to do
	// with a bug in the way lines longer than the given width are wrapped.
	//
	//     https://github.com/muesli/reflow/issues/43
	//
	// TODO: solve this upstream in Glamour/Reflow.
	return wrap.String(mdt, w), nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
