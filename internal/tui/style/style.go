package style

import (
	"github.com/charmbracelet/lipgloss"
)

// XXX: For now, this is in its own package so that it can be shared between
// different packages without incurring an illegal import cycle.

// Styles defines styles for the TUI.
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
	Branch      lipgloss.Style
	HelpKey     lipgloss.Style
	HelpValue   lipgloss.Style
	HelpDivider lipgloss.Style

	Error      lipgloss.Style
	ErrorTitle lipgloss.Style
	ErrorBody  lipgloss.Style

	LogItemSelector   lipgloss.Style
	LogItemActive     lipgloss.Style
	LogItemInactive   lipgloss.Style
	LogItemHash       lipgloss.Style
	LogCommit         lipgloss.Style
	LogCommitHash     lipgloss.Style
	LogCommitAuthor   lipgloss.Style
	LogCommitDate     lipgloss.Style
	LogCommitBody     lipgloss.Style
	LogCommitStatsAdd lipgloss.Style
	LogCommitStatsDel lipgloss.Style

	RefItemSelector lipgloss.Style
	RefItemActive   lipgloss.Style
	RefItemInactive lipgloss.Style
	RefItemBranch   lipgloss.Style
	RefItemTag      lipgloss.Style

	TreeItemSelector lipgloss.Style
	TreeItemActive   lipgloss.Style
	TreeItemInactive lipgloss.Style
	TreeFileDir      lipgloss.Style
	TreeFileMode     lipgloss.Style
	TreeFileSize     lipgloss.Style
	TreeFileContent  lipgloss.Style

	Spinner lipgloss.Style
}

// DefaultStyles returns default styles for the TUI.
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
		Padding(0, 2)

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

	s.Branch = lipgloss.NewStyle().
		Foreground(lipgloss.Color("203")).
		Background(lipgloss.Color("236")).
		Padding(0, 1)

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

	s.LogItemInactive = lipgloss.NewStyle().
		MarginLeft(1)

	s.LogItemSelector = s.LogItemInactive.Copy().
		Width(1).
		Foreground(lipgloss.Color("#B083EA"))

	s.LogItemActive = s.LogItemInactive.Copy().
		Bold(true)

	s.LogItemHash = s.LogItemInactive.Copy().
		Width(7).
		Foreground(lipgloss.Color("#A3A322"))

	s.LogCommit = lipgloss.NewStyle().
		Margin(0, 2)

	s.LogCommitHash = s.LogItemHash.Copy().
		UnsetMarginLeft().
		UnsetWidth().
		Bold(true)

	s.LogCommitBody = lipgloss.NewStyle().
		MarginTop(1).
		MarginLeft(2)

	s.LogCommitStatsAdd = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00D787")).
		Bold(true)

	s.LogCommitStatsDel = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FD5B5B")).
		Bold(true)

	s.RefItemSelector = s.LogItemSelector.Copy()

	s.RefItemActive = s.LogItemActive.Copy()

	s.RefItemInactive = s.LogItemInactive.Copy()

	s.RefItemBranch = lipgloss.NewStyle()

	s.RefItemTag = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#A3A322"))

	s.TreeItemSelector = s.LogItemSelector.Copy()

	s.TreeItemActive = s.LogItemActive.Copy()

	s.TreeItemInactive = s.LogItemInactive.Copy()

	s.TreeFileDir = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00AAFF"))

	s.TreeFileMode = s.LogItemInactive.Copy().
		Width(10).
		Foreground(lipgloss.Color("#777777"))

	s.TreeFileSize = s.LogItemInactive.Copy().
		Foreground(lipgloss.Color("252"))

	s.TreeFileContent = lipgloss.NewStyle()

	s.Spinner = lipgloss.NewStyle().
		MarginLeft(1).
		Foreground(lipgloss.Color("205"))

	return s
}
