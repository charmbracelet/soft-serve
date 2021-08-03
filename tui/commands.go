package tui

import (
	"smoothie/tui/bubbles/commits"

	tea "github.com/charmbracelet/bubbletea"
)

type windowMsg struct{}

func (m *Model) windowChangesCmd() tea.Msg {
	w := <-m.windowChanges
	m.width = w.Width
	m.height = w.Height
	return windowMsg{}
}

func (m *Model) getCommitsCmd() tea.Msg {
	m.commitTimeline = commits.NewBubble(m.height, 2, 80, m.repoSource.GetCommits(200))
	m.state = loadedState
	return nil
}
