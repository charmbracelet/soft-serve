package tui

import (
	"github.com/charmbracelet/lipgloss"
)

const boxLeftWidth = 25
const boxRightWidth = 85
const headerHeight = 1
const footerHeight = 2
const appPadding = 1
const boxPadding = 1
const viewportHeightConstant = 7 // TODO figure out why this needs to be 7
const horizontalPadding = appPadding * 2
const verticalPadding = headerHeight + footerHeight + (appPadding * 2)

var appBoxStyle = lipgloss.NewStyle().
	PaddingLeft(appPadding).
	PaddingRight(appPadding)

var inactiveBoxStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#606060")).
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("#303030")).
	Padding(boxPadding)

var activeBoxStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#FFFFFF")).
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("#714C7B")).
	Padding(boxPadding)

var headerStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#714C7B")).
	Align(lipgloss.Right).
	Bold(true)

var normalStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#FFFFFF"))

var footerStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#373737"))

var footerHighlightStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#DCDCDC"))

var errorStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#FF00000"))
