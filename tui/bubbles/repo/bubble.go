package repo

import (
	"bytes"
	"smoothie/git"
	"text/template"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

const glamourMaxWidth = 120

type ErrMsg struct {
	Error error
}

type Bubble struct {
	templateObject interface{}
	repoSource     *git.RepoSource
	name           string
	repo           *git.Repo
	readmeViewport *ViewportBubble
	readme         string
	height         int
	heightMargin   int
	width          int
	widthMargin    int
}

func NewBubble(rs *git.RepoSource, name string, width, wm, height, hm int, tmp interface{}) *Bubble {
	b := &Bubble{
		templateObject: tmp,
		repoSource:     rs,
		name:           name,
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
	b.readmeViewport.Viewport.Height = h - b.heightMargin
}

func (b *Bubble) GotoTop() {
	b.readmeViewport.Viewport.GotoTop()
}

func (b *Bubble) View() string {
	return b.readmeViewport.View()
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
	// TODO: read gaps in appropriate style to remove the magic number below.
	w := b.width - b.widthMargin - 2
	if w > glamourMaxWidth {
		w = glamourMaxWidth
	}
	tr, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(w),
	)

	if err != nil {
		return "", err
	}
	mdt, err := tr.Render(md)
	if err != nil {
		return "", err
	}
	// Enforce a maximum width for cases when glamour lines run long.
	//
	// TODO: use Reflow's unconditional wrapping to force-wrap long lines. This
	// should utlimately happen as a Glamour option.
	//
	// See:
	// https://github.com/muesli/reflow#unconditional-wrapping
	mdt = lipgloss.NewStyle().MaxWidth(w).Render(mdt)
	return mdt, nil
}
