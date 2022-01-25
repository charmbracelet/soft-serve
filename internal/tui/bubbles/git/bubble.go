package git

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/soft-serve/internal/tui/bubbles/git/about"
	"github.com/charmbracelet/soft-serve/internal/tui/bubbles/git/log"
	"github.com/charmbracelet/soft-serve/internal/tui/bubbles/git/refs"
	"github.com/charmbracelet/soft-serve/internal/tui/bubbles/git/tree"
	"github.com/charmbracelet/soft-serve/internal/tui/bubbles/git/types"
	"github.com/charmbracelet/soft-serve/internal/tui/style"
	"github.com/go-git/go-git/v5/plumbing"
)

const (
	repoNameMaxWidth = 32
)

type pageState int

const (
	aboutPage pageState = iota
	refsPage
	logPage
	treePage
)

type Bubble struct {
	page         pageState
	repo         types.Repo
	height       int
	heightMargin int
	width        int
	widthMargin  int
	style        *style.Styles
	boxes        []tea.Model
}

func NewBubble(repo types.Repo, styles *style.Styles, width, wm, height, hm int) *Bubble {
	b := &Bubble{
		repo:         repo,
		page:         aboutPage,
		width:        width,
		widthMargin:  wm,
		height:       height,
		heightMargin: hm,
		style:        styles,
		boxes:        make([]tea.Model, 4),
	}
	heightMargin := hm + lipgloss.Height(b.headerView())
	b.boxes[aboutPage] = about.NewBubble(repo, b.style, b.width, wm, b.height, heightMargin)
	b.boxes[refsPage] = refs.NewBubble(repo, b.style, b.width, wm, b.height, heightMargin)
	b.boxes[logPage] = log.NewBubble(repo, b.style, width, wm, height, heightMargin)
	b.boxes[treePage] = tree.NewBubble(repo, b.style, width, wm, height, heightMargin)
	return b
}

func (b *Bubble) Init() tea.Cmd {
	return b.setupCmd
}

func (b *Bubble) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "A":
			b.page = aboutPage
		case "R":
			b.page = refsPage
		case "L":
			b.page = logPage
		case "T":
			b.page = treePage
		}
	case tea.WindowSizeMsg:
		b.width = msg.Width
		b.height = msg.Height
		for i, bx := range b.boxes {
			m, cmd := bx.Update(msg)
			b.boxes[i] = m
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}
	m, cmd := b.boxes[b.page].Update(msg)
	b.boxes[b.page] = m
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	return b, tea.Batch(cmds...)
}

func (b *Bubble) Help() []types.HelpEntry {
	h := []types.HelpEntry{}
	if b.page != aboutPage {
		h = append(h, types.HelpEntry{"A", "about"})
	}
	if b.page != refsPage {
		h = append(h, types.HelpEntry{"R", "refs"})
	}
	if b.page != logPage {
		h = append(h, types.HelpEntry{"L", "log"})
	}
	if b.page != treePage {
		h = append(h, types.HelpEntry{"T", "tree"})
	}
	h = append(h, b.boxes[b.page].(types.HelpableBubble).Help()...)
	return h
}

func (b *Bubble) Reference() plumbing.ReferenceName {
	return b.repo.GetReference().Name()
}

func (b *Bubble) headerView() string {
	// TODO better header, tabs?
	return ""
}

func (b *Bubble) View() string {
	header := b.headerView()
	return header + b.boxes[b.page].View()
}

func (b *Bubble) setupCmd() tea.Msg {
	cmds := make([]tea.Cmd, 0)
	for _, bx := range b.boxes {
		if bx != nil {
			initCmd := bx.Init()
			if initCmd != nil {
				msg := initCmd()
				switch msg := msg.(type) {
				case types.ErrMsg:
					return msg
				}
			}
			cmds = append(cmds, initCmd)
		}
	}
	return tea.Batch(cmds...)
}
