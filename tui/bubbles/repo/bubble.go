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
	return b.readmeViewport.Update(msg)
}

func (b *Bubble) View() string {
	return b.readmeViewport.View()
}

func (b *Bubble) setupCmd() tea.Msg {
	r, err := b.repoSource.GetRepo(b.name)
	if err != nil {
		return ErrMsg{err}
	}
	md := r.Readme
	if b.templateObject != nil {
		t, err := template.New("readme").Parse(md)
		if err != nil {
			return ErrMsg{err}
		}
		buf := &bytes.Buffer{}
		err = t.Execute(buf, b.templateObject)
		if err != nil {
			return ErrMsg{err}
		}
		md = buf.String()
	}
	md, err = b.glamourize(md)
	if err != nil {
		return ErrMsg{err}
	}
	b.readmeViewport.Viewport.GotoTop()
	b.readmeViewport.Viewport.SetContent(md)
	return nil
}

func (b *Bubble) templatize(t string) (string, error) {
	return "", nil
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
