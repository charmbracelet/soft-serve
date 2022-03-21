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
	"github.com/gogs/git-module"
)

const (
	repoNameMaxWidth = 32
)

type state int

const (
	aboutState state = iota
	refsState
	logState
	treeState
)

type Bubble struct {
	state        state
	repo         types.Repo
	height       int
	heightMargin int
	width        int
	widthMargin  int
	style        *style.Styles
	boxes        []tea.Model
	ref          *git.Reference
}

func NewBubble(repo types.Repo, styles *style.Styles, width, wm, height, hm int) *Bubble {
	b := &Bubble{
		repo:         repo,
		state:        aboutState,
		width:        width,
		widthMargin:  wm,
		height:       height,
		heightMargin: hm,
		style:        styles,
		boxes:        make([]tea.Model, 4),
		ref:          repo.GetHEAD(),
	}
	heightMargin := hm + lipgloss.Height(b.headerView())
	b.boxes[aboutState] = about.NewBubble(repo, b.style, b.width, wm, b.height, heightMargin)
	b.boxes[refsState] = refs.NewBubble(repo, b.style, b.width, wm, b.height, heightMargin)
	b.boxes[logState] = log.NewBubble(repo, b.style, width, wm, height, heightMargin)
	b.boxes[treeState] = tree.NewBubble(repo, b.style, width, wm, height, heightMargin)
	return b
}

func (b *Bubble) Init() tea.Cmd {
	return b.setupCmd
}

func (b *Bubble) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if b.repo.Name() != "config" {
			switch msg.String() {
			case "R":
				b.state = aboutState
			case "B":
				b.state = refsState
			case "C":
				b.state = logState
			case "F":
				b.state = treeState
			}
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
	case refs.RefMsg:
		b.state = treeState
		b.ref = msg
		for i, bx := range b.boxes {
			m, cmd := bx.Update(msg)
			b.boxes[i] = m
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}
	m, cmd := b.boxes[b.state].Update(msg)
	b.boxes[b.state] = m
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	return b, tea.Batch(cmds...)
}

func (b *Bubble) Help() []types.HelpEntry {
	h := []types.HelpEntry{}
	h = append(h, b.boxes[b.state].(types.BubbleHelper).Help()...)
	if b.repo.Name() != "config" {
		h = append(h, types.HelpEntry{Key: "R", Value: "readme"})
		h = append(h, types.HelpEntry{Key: "F", Value: "files"})
		h = append(h, types.HelpEntry{Key: "C", Value: "commits"})
		h = append(h, types.HelpEntry{Key: "B", Value: "branches"})
	}
	return h
}

func (b *Bubble) Reference() *git.Reference {
	return b.ref
}

func (b *Bubble) headerView() string {
	// TODO better header, tabs?
	return ""
}

func (b *Bubble) View() string {
	header := b.headerView()
	return header + b.boxes[b.state].View()
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
