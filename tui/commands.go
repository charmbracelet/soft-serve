package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

type windowMsg struct{}

func (m *Model) windowChangesCmd() tea.Msg {
	w := <-m.windowChanges
	m.width = w.Width
	m.height = w.Height
	return windowMsg{}
}
