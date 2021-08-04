package tui

import (
	"fmt"
	"smoothie/git"
	"smoothie/tui/bubbles/commits"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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

type Model struct {
	state          sessionState
	error          string
	info           string
	width          int
	height         int
	windowChanges  <-chan ssh.Window
	repoSource     *git.RepoSource
	commitTimeline *commits.Bubble
}

func NewModel(width int, height int, windowChanges <-chan ssh.Window, repoSource *git.RepoSource) *Model {
	m := &Model{
		width:         width,
		height:        height,
		windowChanges: windowChanges,
		repoSource:    repoSource,
	}
	m.state = startState
	return m
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.windowChangesCmd, m.loadGitCmd)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	// Always allow state, error, info, window resize and quit messages
	switch msg := msg.(type) {
	case stateMsg:
		m.state = msg.state
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j", "k", "up", "down":
			_, cmd := m.commitTimeline.Update(msg)
			cmds = append(cmds, cmd)
		}
	case errMsg:
		m.error = msg.Error()
		m.state = errorState
		return m, nil
	case infoMsg:
		m.info = msg.text
	case windowMsg:
		cmds = append(cmds, m.windowChangesCmd)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.commitTimeline.Height = msg.Height
	}
	return m, tea.Batch(cmds...)
}

func (m *Model) View() string {
	pad := 6
	h := headerStyle.Width(m.width - pad).Render("Charm Beta")
	f := footerStyle.Render(m.info)
	s := ""
	content := ""
	switch m.state {
	case loadedState:
		s += m.commitTimeline.View()
	case errorState:
		s += errorStyle.Render(fmt.Sprintf("Bummer: %s", m.error))
	default:
		s = normalStyle.Render(fmt.Sprintf("Doing something weird %d", m.state))
	}
	content = h + "\n" + s + "\n" + f
	return appBoxStyle.Render(content)
}

func SessionHandler(reposPath string) func(ssh.Session) (tea.Model, []tea.ProgramOption) {
	rs := git.NewRepoSource(reposPath, time.Second*10)
	return func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
		if len(s.Command()) == 0 {
			pty, changes, active := s.Pty()
			if !active {
				return nil, nil
			}
			return NewModel(pty.Window.Width, pty.Window.Height, changes, rs), []tea.ProgramOption{tea.WithAltScreen()}
		}
		return nil, nil
	}
}
