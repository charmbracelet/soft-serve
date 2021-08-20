package tui

import (
	"fmt"
	"smoothie/git"
	"smoothie/tui/bubbles/repo"
	"smoothie/tui/bubbles/selection"
	"smoothie/tui/style"
	"strings"

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
	Width       int
	Height      int
	InitialRepo string
}

type Bubble struct {
	config        *Config
	styles        *style.Styles
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
}

func NewBubble(cfg *Config, sCfg *SessionConfig) *Bubble {
	b := &Bubble{
		config:      cfg,
		styles:      style.DefaultStyles(),
		width:       sCfg.Width,
		height:      sCfg.Height,
		repoSource:  cfg.RepoSource,
		repoMenu:    make([]MenuEntry, 0),
		boxes:       make([]tea.Model, 2),
		initialRepo: sCfg.InitialRepo,
	}
	b.state = startState
	return b
}

func (b *Bubble) Init() tea.Cmd {
	return b.setupCmd
}

func (b *Bubble) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	// Always allow state, error, info, window resize and quit messages
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return b, tea.Quit
		case "tab", "shift+tab":
			b.activeBox = (b.activeBox + 1) % 2
		case "h", "left":
			if b.activeBox > 0 {
				b.activeBox--
			}
		case "l", "right":
			if b.activeBox < len(b.boxes)-1 {
				b.activeBox++
			}
		}
	case errMsg:
		b.error = msg.Error()
		b.state = errorState
		return b, nil
	case tea.WindowSizeMsg:
		b.width = msg.Width
		b.height = msg.Height
		if b.state == loadedState {
			for i, bx := range b.boxes {
				m, cmd := bx.Update(msg)
				b.boxes[i] = m
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		}
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

func (b *Bubble) viewForBox(i int) string {
	isActive := i == b.activeBox
	switch box := b.boxes[i].(type) {
	case *selection.Bubble:
		var s lipgloss.Style
		s = b.styles.Menu
		if isActive {
			s = s.Copy().BorderForeground(b.styles.ActiveBorderColor)
		}
		return s.Render(box.View())
	case *repo.Bubble:
		box.Active = isActive
		return box.View()
	default:
		panic(fmt.Sprintf("unknown box type %T", box))
	}
}

func (b Bubble) headerView() string {
	w := b.width - b.styles.App.GetHorizontalFrameSize()
	return b.styles.Header.Copy().Width(w).Render(b.config.Name)
}

func (b Bubble) footerView() string {
	w := &strings.Builder{}
	var h []helpEntry
	switch b.state {
	case errorState:
		h = []helpEntry{{"q", "quit"}}
	default:
		h = []helpEntry{
			{"tab", "section"},
			{"↑/↓", "navigate"},
			{"q", "quit"},
		}
		if _, ok := b.boxes[b.activeBox].(*repo.Bubble); ok {
			h = append(h[:2], helpEntry{"f/b", "pgup/pgdown"}, h[2])
		}
	}
	for i, v := range h {
		fmt.Fprint(w, v.Render(b.styles))
		if i != len(h)-1 {
			fmt.Fprint(w, b.styles.HelpDivider)
		}
	}
	return b.styles.Footer.Copy().Width(b.width).Render(w.String())
}

func (b Bubble) errorView() string {
	s := b.styles
	str := lipgloss.JoinHorizontal(
		lipgloss.Top,
		s.ErrorTitle.Render("Bummer"),
		s.ErrorBody.Render(b.error),
	)
	h := b.height -
		s.App.GetVerticalFrameSize() -
		lipgloss.Height(b.headerView()) -
		lipgloss.Height(b.footerView()) -
		s.RepoBody.GetVerticalFrameSize() +
		3 // TODO: this is repo header height -- get it dynamically
	return s.Error.Copy().Height(h).Render(str)
}

func (b Bubble) View() string {
	s := strings.Builder{}
	s.WriteString(b.headerView())
	s.WriteRune('\n')
	switch b.state {
	case loadedState:
		lb := b.viewForBox(0)
		rb := b.viewForBox(1)
		s.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, lb, rb))
	case errorState:
		s.WriteString(b.errorView())
	}
	s.WriteRune('\n')
	s.WriteString(b.footerView())
	return b.styles.App.Render(s.String())
}

type helpEntry struct {
	key string
	val string
}

func (h helpEntry) Render(s *style.Styles) string {
	return fmt.Sprintf("%s %s", s.HelpKey.Render(h.key), s.HelpValue.Render(h.val))
}
