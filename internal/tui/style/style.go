package style

import (
	"github.com/charmbracelet/lipgloss"
)

// XXX: For now, this is in its own package so that it can be shared between
// different packages without incurring an illegal import cycle.

type Styles struct {
	ActiveBorderColor   lipgloss.Color
	InactiveBorderColor lipgloss.Color

	App    lipgloss.Style
	Header lipgloss.Style

	Menu             lipgloss.Style
	MenuCursor       lipgloss.Style
	MenuItem         lipgloss.Style
	SelectedMenuItem lipgloss.Style

	RepoTitleBorder lipgloss.Border
	RepoNoteBorder  lipgloss.Border
	RepoBodyBorder  lipgloss.Border

	RepoTitle    lipgloss.Style
	RepoTitleBox lipgloss.Style
	RepoNote     lipgloss.Style
	RepoNoteBox  lipgloss.Style
	RepoBody     lipgloss.Style

	Footer      lipgloss.Style
	HelpKey     lipgloss.Style
	HelpValue   lipgloss.Style
	HelpDivider lipgloss.Style

	Error      lipgloss.Style
	ErrorTitle lipgloss.Style
	ErrorBody  lipgloss.Style
}

func DefaultStyles() *Styles {
	s := new(Styles)

	s.ActiveBorderColor = lipgloss.Color("62")
	s.InactiveBorderColor = lipgloss.Color("236")

	s.App = lipgloss.NewStyle().
		Margin(1, 2)

	s.Header = lipgloss.NewStyle().
		Foreground(lipgloss.Color("62")).
		Align(lipgloss.Right).
		Bold(true)

	s.Menu = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(s.InactiveBorderColor).
		Padding(1, 2).
		MarginRight(1).
		Width(24)

	s.MenuCursor = lipgloss.NewStyle().
		Foreground(lipgloss.Color("213")).
		SetString(">")

	s.MenuItem = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		PaddingLeft(2)

	s.SelectedMenuItem = lipgloss.NewStyle().
		Foreground(lipgloss.Color("207")).
		PaddingLeft(1)

	s.RepoTitleBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "┬",
		BottomLeft:  "├",
		BottomRight: "┴",
	}

	s.RepoNoteBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "┬",
		TopRight:    "╮",
		BottomLeft:  "┴",
		BottomRight: "┤",
	}

	s.RepoBodyBorder = lipgloss.Border{
		Top:         "",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "",
		TopRight:    "",
		BottomLeft:  "╰",
		BottomRight: "╯",
	}

	s.RepoTitle = lipgloss.NewStyle().
		Padding(0, 2).
		Foreground(lipgloss.Color("252"))

	s.RepoTitleBox = lipgloss.NewStyle().
		BorderStyle(s.RepoTitleBorder).
		BorderForeground(s.InactiveBorderColor)

	s.RepoNote = lipgloss.NewStyle().
		Padding(0, 2).
		Foreground(lipgloss.Color("168"))

	s.RepoNoteBox = lipgloss.NewStyle().
		BorderStyle(s.RepoNoteBorder).
		BorderForeground(s.InactiveBorderColor).
		BorderTop(true).
		BorderRight(true).
		BorderBottom(true).
		BorderLeft(false)

	s.RepoBody = lipgloss.NewStyle().
		BorderStyle(s.RepoBodyBorder).
		BorderForeground(s.InactiveBorderColor).
		PaddingRight(1)

	s.Footer = lipgloss.NewStyle().
		MarginTop(1)

	s.HelpKey = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	s.HelpValue = lipgloss.NewStyle().
		Foreground(lipgloss.Color("239"))

	s.HelpDivider = lipgloss.NewStyle().
		Foreground(lipgloss.Color("237")).
		SetString(" • ")

	s.Error = lipgloss.NewStyle().
		Padding(1)

	s.ErrorTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("204")).
		Bold(true).
		Padding(0, 1)

	s.ErrorBody = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		MarginLeft(2).
		Width(52) // for now

	return s
}
