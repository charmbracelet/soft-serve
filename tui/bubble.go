package tui

import (
	"fmt"
	"smoothie/git"
	"smoothie/tui/bubbles/commits"
	"smoothie/tui/bubbles/selection"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
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

type Model struct {
	state         sessionState
	error         string
	width         int
	height        int
	windowChanges <-chan ssh.Window
	repoSource    *git.RepoSource
	repos         []*git.Repo
	boxes         []tea.Model
	activeBox     int

	repoSelect     *selection.Bubble
	commitsLog     *commits.Bubble
	readmeViewport *ViewportBubble
}

func NewModel(width int, height int, windowChanges <-chan ssh.Window, repoSource *git.RepoSource) *Model {
	m := &Model{
		width:         width,
		height:        height,
		windowChanges: windowChanges,
		repoSource:    repoSource,
		boxes:         make([]tea.Model, 2),
		readmeViewport: &ViewportBubble{
			Viewport: &viewport.Model{
				Width:  boxRightWidth - horizontalPadding - 2,
				Height: height - verticalPadding - viewportHeightConstant,
			},
		},
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
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.activeBox = (m.activeBox + 1) % 2
		}
	case errMsg:
		m.error = msg.Error()
		m.state = errorState
		return m, nil
	case windowMsg:
		cmds = append(cmds, m.windowChangesCmd)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case selection.SelectedMsg:
		rmd := m.repos[msg.Index].Readme
		m.readmeViewport.Viewport.GotoTop()
		m.readmeViewport.Viewport.Height = m.height - verticalPadding - viewportHeightConstant
		m.readmeViewport.Viewport.Width = boxLeftWidth - 2
		m.readmeViewport.Viewport.SetContent(rmd)
		m.boxes[1] = m.readmeViewport
	}
	if m.state == loadedState {
		b, cmd := m.boxes[m.activeBox].Update(msg)
		m.boxes[m.activeBox] = b
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return m, tea.Batch(cmds...)
}

func (m *Model) viewForBox(i int, width int) string {
	var ls lipgloss.Style
	if i == m.activeBox {
		ls = activeBoxStyle.Width(width)
	} else {
		ls = inactiveBoxStyle.Width(width)
	}
	return ls.Render(m.boxes[i].View())
}

func (m *Model) View() string {
	h := headerStyle.Width(m.width - horizontalPadding).Render("Charm Beta")
	f := footerStyle.Render("")
	s := ""
	content := ""
	switch m.state {
	case loadedState:
		lb := m.viewForBox(0, boxLeftWidth)
		rb := m.viewForBox(1, boxRightWidth)
		s += lipgloss.JoinHorizontal(lipgloss.Top, lb, rb)
	case errorState:
		s += errorStyle.Render(fmt.Sprintf("Bummer: %s", m.error))
	default:
		s = normalStyle.Render(fmt.Sprintf("Doing something weird %d", m.state))
	}
	content = h + "\n\n" + s + "\n" + f
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
