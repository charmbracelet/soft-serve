package repo

import (
	"github.com/charmbracelet/lipgloss"
)

var commitBoxStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#FFFFFF")).
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("#670083")).
	Padding(1)
var commitRepoNameStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#8922A5"))
var commitAuthorStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#670083"))
var commitAuthorEmailStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#781194"))
var commitDateStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#781194"))
var commitCommentStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#606060")).
	BorderStyle(lipgloss.Border{Left: ">"}).
	PaddingLeft(1).
	PaddingBottom(0).
	Margin(0)
