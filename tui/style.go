package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var appBoxStyle = lipgloss.NewStyle().
	PaddingLeft(2).
	PaddingRight(2).
	MarginBottom(1)

var inactiveBoxStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#606060")).
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("#303030")).
	Padding(1)

var activeBoxStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#FFFFFF")).
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("#714C7B")).
	Padding(1)

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
