package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var appBoxStyle = lipgloss.NewStyle().
	PaddingLeft(2).
	PaddingRight(2).
	MarginBottom(1)

var headerStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#670083")).
	Align(lipgloss.Right).
	Bold(true)

var normalStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#FFFFFF"))

var footerStyle = lipgloss.NewStyle().
	BorderForeground(lipgloss.Color("#6D6D6D")).
	BorderLeft(true).
	Foreground(lipgloss.Color("#373737")).
	Bold(true)

var errorStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#FF00000"))
