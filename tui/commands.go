package tui

import (
	"smoothie/tui/bubbles/commits"
	"smoothie/tui/bubbles/selection"

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
	rs := make([]string, 0)
	for _, r := range m.repoSource.AllRepos() {
		rs = append(rs, r.Name)
	}
	m.bubbles[0] = selection.NewBubble(rs)
	m.bubbles[1] = commits.NewBubble(m.height, 7, 80, m.repoSource.GetCommits(200))
	m.activeBubble = 0
	m.state = loadedState
	return nil
}
