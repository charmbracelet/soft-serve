package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var activeBorderColor = lipgloss.Color("62")
var inactiveBorderColor = lipgloss.Color("236")

var appBoxStyle = lipgloss.NewStyle().
	Margin(1, 2)

var headerStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("62")).
	Align(lipgloss.Right).
	PaddingRight(1).
	Bold(true)

var menuStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(inactiveBorderColor).
	Padding(1, 2).
	MarginRight(1).
	Width(24)

var menuActiveStyle = menuStyle.Copy().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(activeBorderColor)

var menuCursor = lipgloss.NewStyle().
	Foreground(lipgloss.Color("213")).
	SetString(">")

var contentBoxStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(inactiveBorderColor).
	PaddingRight(1).
	MarginBottom(1)

var contentBoxActiveStyle = contentBoxStyle.Copy().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(activeBorderColor)

var menuItemStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("252")).
	PaddingLeft(2)

var selectedMenuItemStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("207")).
	PaddingLeft(1)

var footerStyle = lipgloss.NewStyle()

var helpKeyStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("241"))

var helpValueStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("239"))

var errorStyle = lipgloss.NewStyle().
	Padding(1)

var errorHeaderStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("230")).
	Background(lipgloss.Color("204")).
	Bold(true).
	Padding(0, 1)

var errorBodyStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("252")).
	MarginLeft(2).
	Width(52) // for now

var helpDivider = lipgloss.NewStyle().
	Foreground(lipgloss.Color("237")).
	SetString(" • ")
