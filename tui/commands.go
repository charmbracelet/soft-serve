package tui

import (
	"smoothie/tui/bubbles/commits"
	"smoothie/tui/bubbles/selection"

	tea "github.com/charmbracelet/bubbletea"
)

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
	m.repos = m.repoSource.AllRepos()
	rs := make([]string, 0)
	for _, r := range m.repos {
		rs = append(rs, r.Name)
	}
	m.repoSelect = selection.NewBubble(rs)
	m.boxes[0] = m.repoSelect
	m.commitsLog = commits.NewBubble(
		m.height-verticalPadding-2,
		boxRightWidth-horizontalPadding-2,
		m.repoSource.GetCommits(200),
	)
	m.boxes[1] = m.commitsLog
	m.activeBox = 0
	m.state = loadedState
	return nil
}
