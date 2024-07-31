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

	App                  lipgloss.Style
	ServerName           lipgloss.Style
	TopLevelNormalTab    lipgloss.Style
	TopLevelActiveTab    lipgloss.Style
	TopLevelActiveTabDot lipgloss.Style

	MenuItem       lipgloss.Style
	MenuLastUpdate lipgloss.Style

	RepoSelector struct {
		Normal struct {
			Base    lipgloss.Style
			Title   lipgloss.Style
			Desc    lipgloss.Style
			Command lipgloss.Style
			Updated lipgloss.Style
		}
		Active struct {
			Base    lipgloss.Style
			Title   lipgloss.Style
			Desc    lipgloss.Style
			Command lipgloss.Style
			Updated lipgloss.Style
		}
	}

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
	URLStyle    lipgloss.Style

	Error      lipgloss.Style
	ErrorTitle lipgloss.Style
	ErrorBody  lipgloss.Style

	LogItem struct {
		Normal struct {
			Base    lipgloss.Style
			Hash    lipgloss.Style
			Title   lipgloss.Style
			Desc    lipgloss.Style
			Keyword lipgloss.Style
		}
		Active struct {
			Base    lipgloss.Style
			Hash    lipgloss.Style
			Title   lipgloss.Style
			Desc    lipgloss.Style
			Keyword lipgloss.Style
		}
	}

	Log struct {
		Commit         lipgloss.Style
		CommitHash     lipgloss.Style
		CommitAuthor   lipgloss.Style
		CommitDate     lipgloss.Style
		CommitBody     lipgloss.Style
		CommitStatsAdd lipgloss.Style
		CommitStatsDel lipgloss.Style
		Paginator      lipgloss.Style
	}

	Ref struct {
		Normal struct {
			Base     lipgloss.Style
			Item     lipgloss.Style
			ItemTag  lipgloss.Style
			ItemDesc lipgloss.Style
			ItemHash lipgloss.Style
		}
		Active struct {
			Base     lipgloss.Style
			Item     lipgloss.Style
			ItemTag  lipgloss.Style
			ItemDesc lipgloss.Style
			ItemHash lipgloss.Style
		}
		ItemSelector lipgloss.Style
		Paginator    lipgloss.Style
		Selector     lipgloss.Style
	}

	Tree struct {
		Normal struct {
			FileName lipgloss.Style
			FileDir  lipgloss.Style
			FileMode lipgloss.Style
			FileSize lipgloss.Style
		}
		Active struct {
			FileName lipgloss.Style
			FileDir  lipgloss.Style
			FileMode lipgloss.Style
			FileSize lipgloss.Style
		}
		Selector    lipgloss.Style
		FileContent lipgloss.Style
		Paginator   lipgloss.Style
		Blame       struct {
			Hash    lipgloss.Style
			Message lipgloss.Style
			Who     lipgloss.Style
		}
	}

	Stash struct {
		Normal struct {
			Message lipgloss.Style
		}
		Active struct {
			Message lipgloss.Style
		}
		Title    lipgloss.Style
		Selector lipgloss.Style
	}

	Spinner          lipgloss.Style
	SpinnerContainer lipgloss.Style

	NoContent lipgloss.Style

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

	Code struct {
		LineDigit lipgloss.Style
		LineBar   lipgloss.Style
	}
}

// DefaultStyles returns default styles for the UI.
func DefaultStyles(r *lipgloss.Renderer) *Styles {
	highlightColor := lipgloss.Color("210")
	highlightColorDim := lipgloss.Color("174")
	selectorColor := lipgloss.Color("167")
	hashColor := lipgloss.Color("185")

	s := new(Styles)

	s.ActiveBorderColor = lipgloss.Color("62")
	s.InactiveBorderColor = lipgloss.Color("241")

	s.App = r.NewStyle().
		Margin(1, 2)

	s.ServerName = r.NewStyle().
		Height(1).
		MarginLeft(1).
		MarginBottom(1).
		Padding(0, 1).
		Background(lipgloss.Color("57")).
		Foreground(lipgloss.Color("229")).
		Bold(true)

	s.TopLevelNormalTab = r.NewStyle().
		MarginRight(2)

	s.TopLevelActiveTab = s.TopLevelNormalTab.
		Foreground(lipgloss.Color("36"))

	s.TopLevelActiveTabDot = r.NewStyle().
		Foreground(lipgloss.Color("36"))

	s.RepoSelector.Normal.Base = r.NewStyle().
		PaddingLeft(1).
		Border(lipgloss.Border{Left: " "}, false, false, false, true).
		Height(3)

	s.RepoSelector.Normal.Title = r.NewStyle().Bold(true)

	s.RepoSelector.Normal.Desc = r.NewStyle().
		Foreground(lipgloss.Color("243"))

	s.RepoSelector.Normal.Command = r.NewStyle().
		Foreground(lipgloss.Color("132"))

	s.RepoSelector.Normal.Updated = r.NewStyle().
		Foreground(lipgloss.Color("243"))

	s.RepoSelector.Active.Base = s.RepoSelector.Normal.Base.
		BorderStyle(lipgloss.Border{Left: "┃"}).
		BorderForeground(lipgloss.Color("176"))

	s.RepoSelector.Active.Title = s.RepoSelector.Normal.Title.
		Foreground(lipgloss.Color("212"))

	s.RepoSelector.Active.Desc = s.RepoSelector.Normal.Desc.
		Foreground(lipgloss.Color("246"))

	s.RepoSelector.Active.Updated = s.RepoSelector.Normal.Updated.
		Foreground(lipgloss.Color("212"))

	s.RepoSelector.Active.Command = s.RepoSelector.Normal.Command.
		Foreground(lipgloss.Color("204"))

	s.MenuItem = r.NewStyle().
		PaddingLeft(1).
		Border(lipgloss.Border{
			Left: " ",
		}, false, false, false, true).
		Height(3)

	s.MenuLastUpdate = r.NewStyle().
		Foreground(lipgloss.Color("241")).
		Align(lipgloss.Right)

	s.Repo.Base = r.NewStyle()

	s.Repo.Title = r.NewStyle().
		Padding(0, 2)

	s.Repo.Command = r.NewStyle().
		Foreground(lipgloss.Color("168"))

	s.Repo.Body = r.NewStyle().
		Margin(1, 0)

	s.Repo.Header = r.NewStyle().
		MaxHeight(2).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(lipgloss.Color("236"))

	s.Repo.HeaderName = r.NewStyle().
		Foreground(lipgloss.Color("212")).
		Bold(true)

	s.Repo.HeaderDesc = r.NewStyle().
		Foreground(lipgloss.Color("243"))

	s.Footer = r.NewStyle().
		MarginTop(1).
		Padding(0, 1).
		Height(1)

	s.Branch = r.NewStyle().
		Foreground(lipgloss.Color("203")).
		Background(lipgloss.Color("236")).
		Padding(0, 1)

	s.HelpKey = r.NewStyle().
		Foreground(lipgloss.Color("241"))

	s.HelpValue = r.NewStyle().
		Foreground(lipgloss.Color("239"))

	s.HelpDivider = r.NewStyle().
		Foreground(lipgloss.Color("237")).
		SetString(" • ")

	s.URLStyle = r.NewStyle().
		MarginLeft(1).
		Foreground(lipgloss.Color("168"))

	s.Error = r.NewStyle().
		MarginTop(2)

	s.ErrorTitle = r.NewStyle().
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("204")).
		Bold(true).
		Padding(0, 1)

	s.ErrorBody = r.NewStyle().
		Foreground(lipgloss.Color("252")).
		MarginLeft(2)

	s.LogItem.Normal.Base = r.NewStyle().
		Border(lipgloss.Border{
			Left: " ",
		}, false, false, false, true).
		PaddingLeft(1)

	s.LogItem.Active.Base = s.LogItem.Normal.Base.
		Border(lipgloss.Border{
			Left: "┃",
		}, false, false, false, true).
		BorderForeground(selectorColor)

	s.LogItem.Active.Hash = s.LogItem.Normal.Hash.
		Foreground(hashColor)

	s.LogItem.Active.Hash = r.NewStyle().
		Bold(true).
		Foreground(highlightColor)

	s.LogItem.Normal.Title = r.NewStyle().
		Foreground(lipgloss.Color("105"))

	s.LogItem.Active.Title = r.NewStyle().
		Foreground(highlightColor).
		Bold(true)

	s.LogItem.Normal.Desc = r.NewStyle().
		Foreground(lipgloss.Color("246"))

	s.LogItem.Active.Desc = r.NewStyle().
		Foreground(lipgloss.Color("95"))

	s.LogItem.Active.Keyword = s.LogItem.Active.Desc.
		Foreground(highlightColorDim)

	s.LogItem.Normal.Hash = r.NewStyle().
		Foreground(hashColor)

	s.LogItem.Active.Hash = r.NewStyle().
		Foreground(highlightColor)

	s.Log.Commit = r.NewStyle().
		Margin(0, 2)

	s.Log.CommitHash = r.NewStyle().
		Foreground(hashColor).
		Bold(true)

	s.Log.CommitBody = r.NewStyle().
		MarginTop(1).
		MarginLeft(2)

	s.Log.CommitStatsAdd = r.NewStyle().
		Foreground(lipgloss.Color("42")).
		Bold(true)

	s.Log.CommitStatsDel = r.NewStyle().
		Foreground(lipgloss.Color("203")).
		Bold(true)

	s.Log.Paginator = r.NewStyle().
		Margin(0).
		Align(lipgloss.Center)

	s.Ref.Normal.Item = r.NewStyle()

	s.Ref.ItemSelector = r.NewStyle().
		Foreground(selectorColor).
		SetString("> ")

	s.Ref.Active.Item = r.NewStyle().
		Foreground(highlightColorDim)

	s.Ref.Normal.Base = r.NewStyle()

	s.Ref.Active.Base = r.NewStyle()

	s.Ref.Normal.ItemTag = r.NewStyle().
		Foreground(lipgloss.Color("39"))

	s.Ref.Active.ItemTag = r.NewStyle().
		Bold(true).
		Foreground(highlightColor)

	s.Ref.Active.Item = r.NewStyle().
		Bold(true).
		Foreground(highlightColor)

	s.Ref.Normal.ItemDesc = r.NewStyle().
		Faint(true)

	s.Ref.Active.ItemDesc = r.NewStyle().
		Foreground(highlightColor).
		Faint(true)

	s.Ref.Normal.ItemHash = r.NewStyle().
		Foreground(hashColor).
		Bold(true)

	s.Ref.Active.ItemHash = r.NewStyle().
		Foreground(highlightColor).
		Bold(true)

	s.Ref.Paginator = s.Log.Paginator

	s.Ref.Selector = r.NewStyle()

	s.Tree.Selector = s.Tree.Normal.FileName.
		Width(1).
		Foreground(selectorColor)

	s.Tree.Normal.FileName = r.NewStyle().
		MarginLeft(1)

	s.Tree.Active.FileName = s.Tree.Normal.FileName.
		Bold(true).
		Foreground(highlightColor)

	s.Tree.Normal.FileDir = r.NewStyle().
		Foreground(lipgloss.Color("39"))

	s.Tree.Active.FileDir = r.NewStyle().
		Foreground(highlightColor)

	s.Tree.Normal.FileMode = s.Tree.Active.FileName.
		Width(10).
		Foreground(lipgloss.Color("243"))

	s.Tree.Active.FileMode = s.Tree.Normal.FileMode.
		Foreground(highlightColorDim)

	s.Tree.Normal.FileSize = s.Tree.Normal.FileName.
		Foreground(lipgloss.Color("243"))

	s.Tree.Active.FileSize = s.Tree.Normal.FileName.
		Foreground(highlightColorDim)

	s.Tree.FileContent = r.NewStyle()

	s.Tree.Paginator = s.Log.Paginator

	s.Tree.Blame.Hash = r.NewStyle().
		Foreground(hashColor).
		Bold(true)

	s.Tree.Blame.Message = r.NewStyle()

	s.Tree.Blame.Who = r.NewStyle().
		Faint(true)

	s.Spinner = r.NewStyle().
		MarginTop(1).
		MarginLeft(2).
		Foreground(lipgloss.Color("205"))

	s.SpinnerContainer = r.NewStyle()

	s.NoContent = r.NewStyle().
		MarginTop(1).
		MarginLeft(2).
		Foreground(lipgloss.Color("242"))

	s.StatusBar = r.NewStyle().
		Height(1)

	s.StatusBarKey = r.NewStyle().
		Bold(true).
		Padding(0, 1).
		Background(lipgloss.Color("206")).
		Foreground(lipgloss.Color("228"))

	s.StatusBarValue = r.NewStyle().
		Padding(0, 1).
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("243"))

	s.StatusBarInfo = r.NewStyle().
		Padding(0, 1).
		Background(lipgloss.Color("212")).
		Foreground(lipgloss.Color("230"))

	s.StatusBarBranch = r.NewStyle().
		Padding(0, 1).
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230"))

	s.StatusBarHelp = r.NewStyle().
		Padding(0, 1).
		Background(lipgloss.Color("237")).
		Foreground(lipgloss.Color("243"))

	s.Tabs = r.NewStyle().
		Height(1)

	s.TabInactive = r.NewStyle()

	s.TabActive = r.NewStyle().
		Underline(true).
		Foreground(lipgloss.Color("36"))

	s.TabSeparator = r.NewStyle().
		SetString("│").
		Padding(0, 1).
		Foreground(lipgloss.Color("238"))

	s.Code.LineDigit = r.NewStyle().Foreground(lipgloss.Color("239"))

	s.Code.LineBar = r.NewStyle().Foreground(lipgloss.Color("236"))

	s.Stash.Normal.Message = r.NewStyle().MarginLeft(1)

	s.Stash.Active.Message = s.Stash.Normal.Message.Foreground(selectorColor)

	s.Stash.Title = r.NewStyle().
		Foreground(hashColor).
		Bold(true)

	s.Stash.Selector = r.NewStyle().
		Width(1).
		Foreground(selectorColor)

	return s
}
