package tui

import (
	"smoothie/tui/bubbles/commits"

	tea "github.com/charmbracelet/bubbletea"
)

type stateMsg struct{ state sessionState }
type infoMsg struct{ text string }
type windowMsg struct{}
type errMsg struct{ err error }

func (e errMsg) Error() string {
	return e.err.Error()
}

func (m *Model) windowChangesCmd() tea.Msg {
	w := <-m.windowChanges
	m.width = w.Width
	m.height = w.Height
	return windowMsg{}
}

func (m *Model) loadGitCmd() tea.Msg {
	m.commitTimeline = commits.NewBubble(m.height, 2, 80, m.repoSource.GetCommits(200))
	m.state = loadedState
	return nil
}
