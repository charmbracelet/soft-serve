package utils

import tea "github.com/charmbracelet/bubbletea"

type BubbleReset interface {
	Reset() tea.Msg
}
