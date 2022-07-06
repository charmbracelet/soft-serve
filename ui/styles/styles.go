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

	Repo struct {
		Base       lipgloss.Style
		Title      lipgloss.Style
		Command    lipgloss.Style
		Body       lipgloss.Style
		Header     lipgloss.Style
		HeaderName lipgloss.Style
		HeaderDesc lipgloss.Style
	}

	Footer      lipgloss.Style
	Branch      lipgloss.Style
	HelpKey     lipgloss.Style
	HelpValue   lipgloss.Style
	HelpDivider lipgloss.Style

	Error      lipgloss.Style
	ErrorTitle lipgloss.Style
	ErrorBody  lipgloss.Style

	AboutNoReadme lipgloss.Style

	Log struct {
		Item                lipgloss.Style
		ItemSelector        lipgloss.Style
		ItemActive          lipgloss.Style
		ItemInactive        lipgloss.Style
		ItemHash            lipgloss.Style
		ItemTitleInactive   lipgloss.Style
		ItemTitleActive     lipgloss.Style
		ItemDescInactive    lipgloss.Style
		ItemDescActive      lipgloss.Style
		ItemKeywordActive   lipgloss.Style
		ItemKeywordInactive lipgloss.Style
		Commit              lipgloss.Style
		CommitHash          lipgloss.Style
		CommitAuthor        lipgloss.Style
		CommitDate          lipgloss.Style
		CommitBody          lipgloss.Style
		CommitStatsAdd      lipgloss.Style
		CommitStatsDel      lipgloss.Style
		Paginator           lipgloss.Style
	}

	Ref struct {
		ItemSelector    lipgloss.Style
		ItemActive      lipgloss.Style
		ItemInactive    lipgloss.Style
		ItemBranch      lipgloss.Style
		ItemTagInactive lipgloss.Style
		ItemTagActive   lipgloss.Style
		Paginator       lipgloss.Style
	}

	Tree struct {
		ItemSelector     lipgloss.Style
		ItemActive       lipgloss.Style
		ItemInactive     lipgloss.Style
		FileDirInactive  lipgloss.Style
		FileDirActive    lipgloss.Style
		FileModeInactive lipgloss.Style
		FileModeActive   lipgloss.Style
		FileSizeInactive lipgloss.Style
		FileSizeActive   lipgloss.Style
		FileContent      lipgloss.Style
		Paginator        lipgloss.Style
		NoItems          lipgloss.Style
	}

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
	selectorColor := lipgloss.Color("167")

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

	s.Repo.Base = lipgloss.NewStyle()

	s.Repo.Title = lipgloss.NewStyle().
		Padding(0, 2)

	s.Repo.Command = lipgloss.NewStyle().
		Foreground(lipgloss.Color("168"))

	s.Repo.Body = lipgloss.NewStyle().
		Margin(1, 0)

	s.Repo.Header = lipgloss.NewStyle().
		Height(2).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(lipgloss.Color("238"))

	s.Repo.HeaderName = lipgloss.NewStyle().
		Bold(true)

	s.Repo.HeaderDesc = lipgloss.NewStyle().
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
		Foreground(lipgloss.Color("242"))

	s.Log.ItemInactive = lipgloss.NewStyle().
		Border(lipgloss.Border{
			Left: " ",
		}, false, false, false, true).
		PaddingLeft(1)

	s.Log.ItemActive = s.Log.ItemInactive.Copy().
		Border(lipgloss.Border{
			Left: "┃",
		}, false, false, false, true).
		BorderForeground(selectorColor)

	s.Log.ItemSelector = s.Log.ItemInactive.Copy().
		Width(1).
		Foreground(lipgloss.Color("62"))

	s.Log.ItemHash = s.Log.ItemInactive.Copy().
		Foreground(lipgloss.Color("184"))

	s.Log.ItemTitleInactive = lipgloss.NewStyle().
		Foreground(lipgloss.Color("105"))

	s.Log.ItemTitleActive = lipgloss.NewStyle().
		Foreground(highlightColor).
		Bold(true)

	s.Log.ItemDescInactive = lipgloss.NewStyle().
		Foreground(lipgloss.Color("246"))

	s.Log.ItemDescActive = lipgloss.NewStyle().
		Foreground(lipgloss.Color("95"))

	s.Log.ItemKeywordActive = s.Log.ItemDescActive.Copy().
		Foreground(highlightColorDim)

	s.Log.Commit = lipgloss.NewStyle().
		Margin(0, 2)

	s.Log.CommitHash = lipgloss.NewStyle().
		Foreground(lipgloss.Color("184")).
		Bold(true)

	s.Log.CommitBody = lipgloss.NewStyle().
		MarginTop(1).
		MarginLeft(2)

	s.Log.CommitStatsAdd = lipgloss.NewStyle().
		Foreground(lipgloss.Color("42")).
		Bold(true)

	s.Log.CommitStatsDel = lipgloss.NewStyle().
		Foreground(lipgloss.Color("203")).
		Bold(true)

	s.Log.Paginator = lipgloss.NewStyle().
		Margin(0).
		Align(lipgloss.Center)

	s.Ref.ItemInactive = lipgloss.NewStyle()

	s.Ref.ItemSelector = lipgloss.NewStyle().
		Foreground(selectorColor).
		SetString("> ")

	s.Ref.ItemActive = s.Ref.ItemActive.Copy().
		Foreground(highlightColorDim)

	s.Ref.ItemBranch = lipgloss.NewStyle()

	s.Ref.ItemTagInactive = lipgloss.NewStyle().
		Foreground(lipgloss.Color("185"))

	s.Ref.ItemTagActive = lipgloss.NewStyle().
		Bold(true).
		Foreground(highlightColor)

	s.Ref.ItemActive = lipgloss.NewStyle().
		Bold(true).
		Foreground(highlightColor)

	s.Ref.Paginator = s.Log.Paginator.Copy()

	s.Tree.ItemSelector = s.Tree.ItemInactive.Copy().
		Width(1).
		Foreground(selectorColor)

	s.Tree.ItemInactive = lipgloss.NewStyle().
		MarginLeft(1)

	s.Tree.ItemActive = s.Tree.ItemInactive.Copy().
		Bold(true).
		Foreground(highlightColor)

	s.Tree.FileDirInactive = lipgloss.NewStyle().
		Foreground(lipgloss.Color("39"))

	s.Tree.FileDirActive = lipgloss.NewStyle().
		Foreground(highlightColor)

	s.Tree.FileModeInactive = s.Tree.ItemInactive.Copy().
		Width(10).
		Foreground(lipgloss.Color("243"))

	s.Tree.FileModeActive = s.Tree.FileModeInactive.Copy().
		Foreground(highlightColorDim)

	s.Tree.FileSizeInactive = s.Tree.ItemInactive.Copy().
		Foreground(lipgloss.Color("243"))

	s.Tree.FileSizeActive = s.Tree.ItemInactive.Copy().
		Foreground(highlightColorDim)

	s.Tree.FileContent = lipgloss.NewStyle()

	s.Tree.Paginator = s.Log.Paginator.Copy()

	s.Tree.NoItems = s.AboutNoReadme.Copy()

	s.Spinner = lipgloss.NewStyle().
		MarginTop(1).
		MarginLeft(2).
		Foreground(lipgloss.Color("205"))

	s.CodeNoContent = lipgloss.NewStyle().
		SetString("No Content.").
		MarginTop(1).
		MarginLeft(2).
		Foreground(lipgloss.Color("242"))

	s.StatusBar = lipgloss.NewStyle().
		Height(1)

	s.StatusBarKey = lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1).
		Background(lipgloss.Color("206")).
		Foreground(lipgloss.Color("228"))

	s.StatusBarValue = lipgloss.NewStyle().
		Padding(0, 1).
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("243"))

	s.StatusBarInfo = lipgloss.NewStyle().
		Padding(0, 1).
		Background(lipgloss.Color("212")).
		Foreground(lipgloss.Color("230"))

	s.StatusBarBranch = lipgloss.NewStyle().
		Padding(0, 1).
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230"))

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
