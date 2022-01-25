package repo

import (
	"fmt"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	gitui "github.com/charmbracelet/soft-serve/internal/tui/bubbles/git"
	gittypes "github.com/charmbracelet/soft-serve/internal/tui/bubbles/git/types"
	"github.com/charmbracelet/soft-serve/internal/tui/style"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/muesli/reflow/truncate"
	"github.com/muesli/reflow/wrap"
)

const (
	repoNameMaxWidth = 32
)

type Bubble struct {
	name         string
	host         string
	port         int
	repo         gittypes.Repo
	styles       *style.Styles
	width        int
	widthMargin  int
	height       int
	heightMargin int
	box          *gitui.Bubble

	Active bool
}

func NewBubble(name, host string, port int, repo gittypes.Repo, styles *style.Styles, width, wm, height, hm int) *Bubble {
	b := &Bubble{
		name:         name,
		host:         host,
		port:         port,
		repo:         repo,
		width:        width,
		widthMargin:  wm,
		height:       height,
		heightMargin: hm,
		styles:       styles,
	}
	b.box = gitui.NewBubble(repo, styles, width, wm+styles.RepoBody.GetHorizontalBorderSize(), height, hm+lipgloss.Height(b.headerView())-styles.RepoBody.GetVerticalBorderSize())
	return b
}

func (b *Bubble) Init() tea.Cmd {
	return b.box.Init()
}

func (b *Bubble) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		b.width = msg.Width
		b.height = msg.Height
	}
	box, cmd := b.box.Update(msg)
	b.box = box.(*gitui.Bubble)
	return b, cmd
}

func (b *Bubble) Help() []gittypes.HelpEntry {
	return b.box.Help()
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
	height := gittypes.Max(lipgloss.Height(title), lipgloss.Height(note))
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
		Render(b.box.View())
	return header + body
}

func (b *Bubble) Reference() plumbing.ReferenceName {
	return b.box.Reference()
}

func (b Bubble) sshAddress() string {
	p := ":" + strconv.Itoa(int(b.port))
	if p == ":22" {
		p = ""
	}
	return fmt.Sprintf("ssh://%s%s/%s", b.host, p, b.name)
}
