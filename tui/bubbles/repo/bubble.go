package repo

import (
	"bytes"
	"smoothie/git"
	"text/template"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
)

type ErrMsg struct {
	Error error
}

type Bubble struct {
	templateObject interface{}
	repoSource     *git.RepoSource
	name           string
	repo           *git.Repo
	readmeViewport *ViewportBubble
}

func NewBubble(rs *git.RepoSource, name string, width int, height int, tmp interface{}) *Bubble {
	return &Bubble{
		templateObject: tmp,
		repoSource:     rs,
		name:           name,
		readmeViewport: &ViewportBubble{
			Viewport: &viewport.Model{
				Width:  width,
				Height: height,
			},
		},
	}
}

func (b *Bubble) Init() tea.Cmd {
	return b.setupCmd
}

func (b *Bubble) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	rv, cmd := b.readmeViewport.Update(msg)
	b.readmeViewport = rv.(*ViewportBubble)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	return b, tea.Batch(cmds...)
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
	md, err = b.glamourize(md)
	if err != nil {
		return ErrMsg{err}
	}
	b.GotoTop()
	b.readmeViewport.Viewport.SetContent(md)
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
	tr, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(b.readmeViewport.Viewport.Width),
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
