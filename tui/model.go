package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gliderlabs/ssh"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type sessionState int

const (
	startState sessionState = iota
	errorState
	commitsLoadedState
	quittingState
	quitState
)

type stateMsg struct{ state sessionState }
type infoMsg struct{ text string }
type errMsg struct{ err error }

func (e errMsg) Error() string {
	return e.err.Error()
}

func SessionHandler(repoPath string) func(ssh.Session) (tea.Model, []tea.ProgramOption) {
	rs := newRepoSource(repoPath)
	return func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
		if len(s.Command()) == 0 {
			pty, changes, active := s.Pty()
			if !active {
				return nil, nil
			}
			return NewModel(pty.Window.Width, pty.Window.Height, changes, rs), nil
		}
		return nil, nil
	}
}

type Model struct {
	state         sessionState
	error         string
	info          string
	width         int
	height        int
	windowChanges <-chan ssh.Window
	repos         *repoSource
	commits       []*object.Commit
}

func NewModel(width int, height int, windowChanges <-chan ssh.Window, repos *repoSource) *Model {
	m := &Model{
		width:         width,
		height:        height,
		windowChanges: windowChanges,
		repos:         repos,
		commits:       make([]*object.Commit, 0),
	}
	m.state = startState
	return m
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.windowChangesCmd, tea.HideCursor, m.getCommitsCmd)
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
	case startState:
		s += normalStyle.Render("Loading")
	case commitsLoadedState:
		for _, c := range m.commits {
			msg := fmt.Sprintf("%s %s %s %s", c.Author.When, c.Author.Name, c.Author.Email, c.Message)
			s += normalStyle.Render(msg) + "\n"
		}
	case errorState:
		s += errorStyle.Render(fmt.Sprintf("Bummer: %s", m.error))
	default:
		s = normalStyle.Render(fmt.Sprintf("Doing something weird %d", m.state))
	}
	content = h + "\n" + s + "\n" + f
	return appBoxStyle.Render(content)
}
