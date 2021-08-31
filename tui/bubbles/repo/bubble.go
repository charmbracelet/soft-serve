package repo

import (
	"bytes"
	"fmt"
	"soft-serve/git"
	"soft-serve/tui/style"
	"strconv"
	"text/template"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
	"github.com/muesli/reflow/wrap"
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
	Port int64
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
	b.readmeViewport.Viewport.Height = h - b.heightMargin
}

func (b *Bubble) GotoTop() {
	b.readmeViewport.Viewport.GotoTop()
}

func (b Bubble) headerView() string {
	ts := b.styles.RepoTitle
	ns := b.styles.RepoNote
	if b.Active {
		ts = ts.Copy().BorderForeground(b.styles.ActiveBorderColor)
		ns = ns.Copy().BorderForeground(b.styles.ActiveBorderColor)
	}
	var gc string
	n := b.name
	if n == "config" {
		n = "Home"
	} else {
		gc = fmt.Sprintf("git clone %s", b.sshAddress())
	}
	title := ts.Render(n)
	note := ns.Width(b.width - b.widthMargin - lipgloss.Width(title)).Render(gc)
	return lipgloss.JoinHorizontal(lipgloss.Top, title, note)
}

func (b *Bubble) View() string {
	header := b.headerView()
	bs := b.styles.RepoBody.Copy()
	if b.Active {
		bs = bs.BorderForeground(b.styles.ActiveBorderColor)
	}
	body := bs.
		Width(b.width - b.widthMargin - b.styles.RepoBody.GetVerticalFrameSize()).
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
	// TODO: read gaps in appropriate style to remove the magic number below.
	w := b.width - b.widthMargin - 4
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
	// TODO: This should utlimately be implemented as a Glamour option.
	mdt = wrap.String(wordwrap.String((mdt), w), w)
	return mdt, nil
}
