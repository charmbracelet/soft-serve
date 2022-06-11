package styles

import (
	"github.com/charmbracelet/lipgloss"
)

// XXX: For now, this is in its own package so that it can be shared between
// different packages without incurring an illegal import cycle.

// Styles defines styles for the UI.
type Styles struct {
	ActiveBorderColor   lipgloss.Color
	InactiveBorderColor lipgloss.Color

	App    lipgloss.Style
	Header lipgloss.Style

	Menu           lipgloss.Style
	MenuCursor     lipgloss.Style
	MenuItem       lipgloss.Style
	MenuLastUpdate lipgloss.Style

	// Selection page styles
	SelectorBox lipgloss.Style
	ReadmeBox   lipgloss.Style

	Repo           lipgloss.Style
	RepoTitle      lipgloss.Style
	RepoCommand    lipgloss.Style
	RepoBody       lipgloss.Style
	RepoHeader     lipgloss.Style
	RepoHeaderName lipgloss.Style
	RepoHeaderDesc lipgloss.Style

	Footer      lipgloss.Style
	Branch      lipgloss.Style
	HelpKey     lipgloss.Style
	HelpValue   lipgloss.Style
	HelpDivider lipgloss.Style

	Error      lipgloss.Style
	ErrorTitle lipgloss.Style
	ErrorBody  lipgloss.Style

	AboutNoReadme lipgloss.Style

	LogItem           lipgloss.Style
	LogItemSelector   lipgloss.Style
	LogItemActive     lipgloss.Style
	LogItemInactive   lipgloss.Style
	LogItemHash       lipgloss.Style
	LogItemTitle      lipgloss.Style
	LogCommit         lipgloss.Style
	LogCommitHash     lipgloss.Style
	LogCommitAuthor   lipgloss.Style
	LogCommitDate     lipgloss.Style
	LogCommitBody     lipgloss.Style
	LogCommitStatsAdd lipgloss.Style
	LogCommitStatsDel lipgloss.Style
	LogPaginator      lipgloss.Style

	RefItemSelector lipgloss.Style
	RefItemActive   lipgloss.Style
	RefItemInactive lipgloss.Style
	RefItemBranch   lipgloss.Style
	RefItemTag      lipgloss.Style
	RefPaginator    lipgloss.Style

	TreeItemSelector     lipgloss.Style
	TreeItemActive       lipgloss.Style
	TreeItemInactive     lipgloss.Style
	TreeFileDirInactive  lipgloss.Style
	TreeFileDirActive    lipgloss.Style
	TreeFileModeInactive lipgloss.Style
	TreeFileModeActive   lipgloss.Style
	TreeFileSizeInactive lipgloss.Style
	TreeFileSizeActive   lipgloss.Style
	TreeFileContent      lipgloss.Style
	TreePaginator        lipgloss.Style
	TreeNoItems          lipgloss.Style

	Spinner lipgloss.Style

	CodeNoContent lipgloss.Style

	StatusBar       lipgloss.Style
	StatusBarKey    lipgloss.Style
	StatusBarValue  lipgloss.Style
	StatusBarInfo   lipgloss.Style
	StatusBarBranch lipgloss.Style
	StatusBarHelp   lipgloss.Style

	Tabs         lipgloss.Style
	TabInactive  lipgloss.Style
	TabActive    lipgloss.Style
	TabSeparator lipgloss.Style
}

// DefaultStyles returns default styles for the UI.
func DefaultStyles() *Styles {
	highlightColor := lipgloss.Color("210")
	highlightColorDim := lipgloss.Color("174")

	s := new(Styles)

	s.ActiveBorderColor = lipgloss.Color("62")
	s.InactiveBorderColor = lipgloss.Color("241")

	s.App = lipgloss.NewStyle().
		Margin(1, 2)

	s.Header = lipgloss.NewStyle().
		Align(lipgloss.Left).
		Height(1).
		PaddingLeft(1).
		MarginBottom(1).
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
		PaddingLeft(1).
		Border(lipgloss.Border{
			Left: " ",
		}, false, false, false, true).
		Height(3)

	s.MenuLastUpdate = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Align(lipgloss.Right)

	s.SelectorBox = lipgloss.NewStyle()

	s.ReadmeBox = lipgloss.NewStyle()

	s.Repo = lipgloss.NewStyle()

	s.RepoTitle = lipgloss.NewStyle().
		Padding(0, 2)

	s.RepoCommand = lipgloss.NewStyle().
		Foreground(lipgloss.Color("168"))

	s.RepoBody = lipgloss.NewStyle().
		Margin(1, 0)

	s.RepoHeader = lipgloss.NewStyle().
		Height(2).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(lipgloss.Color("238"))

	s.RepoHeaderName = lipgloss.NewStyle().
		Bold(true)

	s.RepoHeaderDesc = lipgloss.NewStyle().
		Faint(true)

	s.Footer = lipgloss.NewStyle().
		MarginTop(1).
		Padding(0, 1).
		Height(1)

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
		MarginTop(2)

	s.ErrorTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("204")).
		Bold(true).
		Padding(0, 1)

	s.ErrorBody = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		MarginLeft(2)

	s.AboutNoReadme = lipgloss.NewStyle().
		MarginTop(1).
		MarginLeft(2).
		Foreground(lipgloss.Color("#626262"))

	s.LogItemInactive = lipgloss.NewStyle().
		Border(lipgloss.Border{
			Left: " ",
		}, false, false, false, true).
		PaddingLeft(1)

	s.LogItemActive = s.LogItemInactive.Copy().
		Border(lipgloss.Border{
			Left: "┃",
		}, false, false, false, true).
		BorderForeground(lipgloss.Color("#B083EA"))

	s.LogItemSelector = s.LogItemInactive.Copy().
		Width(1).
		Foreground(lipgloss.Color("#B083EA"))

	s.LogItemHash = s.LogItemInactive.Copy().
		Foreground(lipgloss.Color("#A3A322"))

	s.LogItemTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#B083EA"))

	s.LogCommit = lipgloss.NewStyle().
		Margin(0, 2)

	s.LogCommitHash = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#A3A322")).
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

	s.LogPaginator = lipgloss.NewStyle().
		Margin(0).
		Align(lipgloss.Center)

	s.RefItemInactive = lipgloss.NewStyle().
		MarginLeft(1)

	s.RefItemSelector = lipgloss.NewStyle().
		Width(1).
		Foreground(lipgloss.Color("#B083EA"))

	s.RefItemActive = lipgloss.NewStyle().
		MarginLeft(1).
		Bold(true)

	s.RefItemBranch = lipgloss.NewStyle()

	s.RefItemTag = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#A3A322"))

	s.RefPaginator = s.LogPaginator.Copy()

	s.TreeItemSelector = s.TreeItemInactive.Copy().
		Width(1).
		Foreground(lipgloss.Color("#B083EA"))

	s.TreeItemInactive = lipgloss.NewStyle().
		MarginLeft(1)

	s.TreeItemActive = s.TreeItemInactive.Copy().
		Bold(true).
		Foreground(highlightColor)

	s.TreeFileDirInactive = lipgloss.NewStyle().
		Foreground(lipgloss.Color("39"))

	s.TreeFileDirActive = lipgloss.NewStyle().
		Foreground(highlightColor)

	s.TreeFileModeInactive = s.TreeItemInactive.Copy().
		Width(10).
		Foreground(lipgloss.Color("243"))

	s.TreeFileModeActive = s.TreeFileModeInactive.Copy().
		Foreground(highlightColorDim)

	s.TreeFileSizeInactive = s.TreeItemInactive.Copy().
		Foreground(lipgloss.Color("243"))

	s.TreeFileSizeActive = s.TreeItemInactive.Copy().
		Foreground(highlightColorDim)

	s.TreeFileContent = lipgloss.NewStyle()

	s.TreePaginator = s.LogPaginator.Copy()

	s.TreeNoItems = s.AboutNoReadme.Copy()

	s.Spinner = lipgloss.NewStyle().
		MarginTop(1).
		MarginLeft(2).
		Foreground(lipgloss.Color("205"))

	s.CodeNoContent = lipgloss.NewStyle().
		SetString("No Content.").
		MarginTop(1).
		MarginLeft(2).
		Foreground(lipgloss.Color("#626262"))

	s.StatusBar = lipgloss.NewStyle().
		Height(1)

	s.StatusBarKey = lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1).
		Background(lipgloss.Color("#FF5FD2")).
		Foreground(lipgloss.Color("#FFFF87"))

	s.StatusBarValue = lipgloss.NewStyle().
		Padding(0, 1).
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("243"))

	s.StatusBarInfo = lipgloss.NewStyle().
		Padding(0, 1).
		Background(lipgloss.Color("#FF8EC7")).
		Foreground(lipgloss.Color("#F1F1F1"))

	s.StatusBarBranch = lipgloss.NewStyle().
		Padding(0, 1).
		Background(lipgloss.Color("#6E6ED8")).
		Foreground(lipgloss.Color("#F1F1F1"))

	s.StatusBarHelp = lipgloss.NewStyle().
		Padding(0, 1).
		Background(lipgloss.Color("237")).
		Foreground(lipgloss.Color("243"))

	s.Tabs = lipgloss.NewStyle()

	s.TabInactive = lipgloss.NewStyle()

	s.TabActive = lipgloss.NewStyle().
		Foreground(lipgloss.Color("63")).
		Underline(true)

	s.TabSeparator = lipgloss.NewStyle().
		SetString("│").
		Padding(0, 1).
		Foreground(lipgloss.Color("238"))

	return s
}
