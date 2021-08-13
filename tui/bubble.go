package tui

import (
	"fmt"
	"smoothie/git"
	"smoothie/tui/bubbles/commits"
	"smoothie/tui/bubbles/repo"
	"smoothie/tui/bubbles/selection"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gliderlabs/ssh"
)

type sessionState int

const (
	startState sessionState = iota
	errorState
	loadedState
	quittingState
	quitState
)

type Config struct {
	Name         string      `json:"name"`
	Host         string      `json:"host"`
	Port         int64       `json:"port"`
	ShowAllRepos bool        `json:"show_all_repos"`
	Menu         []MenuEntry `json:"menu"`
	RepoSource   *git.RepoSource
}

type MenuEntry struct {
	Name   string `json:"name"`
	Note   string `json:"note"`
	Repo   string `json:"repo"`
	bubble *repo.Bubble
}

type SessionConfig struct {
	Width         int
	Height        int
	WindowChanges <-chan ssh.Window
	InitialRepo   string
}

type Bubble struct {
	config        *Config
	state         sessionState
	error         string
	width         int
	height        int
	windowChanges <-chan ssh.Window
	repoSource    *git.RepoSource
	initialRepo   string
	repoMenu      []MenuEntry
	repos         []*git.Repo
	boxes         []tea.Model
	activeBox     int
	repoSelect    *selection.Bubble
	commitsLog    *commits.Bubble
}

func NewBubble(cfg *Config, sCfg *SessionConfig) *Bubble {
	b := &Bubble{
		config:        cfg,
		width:         sCfg.Width,
		height:        sCfg.Height,
		windowChanges: sCfg.WindowChanges,
		repoSource:    cfg.RepoSource,
		repoMenu:      make([]MenuEntry, 0),
		boxes:         make([]tea.Model, 2),
		initialRepo:   sCfg.InitialRepo,
	}
	b.state = startState
	return b
}

func (b *Bubble) Init() tea.Cmd {
	return tea.Batch(b.windowChangesCmd, b.setupCmd)
}

func (b *Bubble) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	// Always allow state, error, info, window resize and quit messages
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return b, tea.Quit
		case "tab":
			b.activeBox = (b.activeBox + 1) % 2
		}
	case errMsg:
		b.error = msg.Error()
		b.state = errorState
		return b, nil
	case windowMsg:
		cmds = append(cmds, b.windowChangesCmd)
	case tea.WindowSizeMsg:
		b.width = msg.Width
		b.height = msg.Height
	case selection.SelectedMsg:
		b.activeBox = 1
		rb := b.repoMenu[msg.Index].bubble
		rb.GotoTop()
		b.boxes[1] = rb
	case selection.ActiveMsg:
		rb := b.repoMenu[msg.Index].bubble
		rb.GotoTop()
		b.boxes[1] = b.repoMenu[msg.Index].bubble
	}
	if b.state == loadedState {
		ab, cmd := b.boxes[b.activeBox].Update(msg)
		b.boxes[b.activeBox] = ab
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return b, tea.Batch(cmds...)
}

func (b *Bubble) viewForBox(i int, width int) string {
	var ls lipgloss.Style
	if i == b.activeBox {
		ls = activeBoxStyle.Width(width)
	} else {
		ls = inactiveBoxStyle.Width(width)
	}
	return ls.Render(b.boxes[i].View())
}

func (b *Bubble) View() string {
	h := headerStyle.Width(b.width - horizontalPadding).Render(b.config.Name)
	f := footerStyle.Render("")
	s := ""
	content := ""
	switch b.state {
	case loadedState:
		lb := b.viewForBox(0, boxLeftWidth)
		rb := b.viewForBox(1, boxRightWidth)
		s += lipgloss.JoinHorizontal(lipgloss.Top, lb, rb)
	case errorState:
		s += errorStyle.Render(fmt.Sprintf("Bummer: %s", b.error))
	}
	content = h + "\n\n" + s + "\n" + f
	return appBoxStyle.Render(content)
}
