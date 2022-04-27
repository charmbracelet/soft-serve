package repo

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	gansi "github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/lipgloss"
	ggit "github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/ui/common"
	"github.com/charmbracelet/soft-serve/ui/components/selector"
	"github.com/charmbracelet/soft-serve/ui/components/viewport"
	"github.com/charmbracelet/soft-serve/ui/git"
	"github.com/muesli/reflow/wrap"
	"github.com/muesli/termenv"
)

type logView int

const (
	logViewCommits logView = iota
	logViewDiff
)

// LogCountMsg is a message that contains the number of commits in a repo.
type LogCountMsg int64

// LogItemsMsg is a message that contains a slice of LogItem.
type LogItemsMsg []selector.IdentifiableItem

// LogCommitMsg is a message that contains a git commit.
type LogCommitMsg *ggit.Commit

// LogDiffMsg is a message that contains a git diff.
type LogDiffMsg *ggit.Diff

// Log is a model that displays a list of commits and their diffs.
type Log struct {
	common         common.Common
	selector       *selector.Selector
	vp             *viewport.Viewport
	activeView     logView
	repo           git.GitRepo
	ref            *ggit.Reference
	count          int64
	nextPage       int
	activeCommit   *ggit.Commit
	selectedCommit *ggit.Commit
	currentDiff    *ggit.Diff
}

// NewLog creates a new Log model.
func NewLog(common common.Common) *Log {
	l := &Log{
		common:     common,
		vp:         viewport.New(),
		activeView: logViewCommits,
	}
	selector := selector.New(common, []selector.IdentifiableItem{}, LogItemDelegate{common.Styles})
	selector.SetShowFilter(false)
	selector.SetShowHelp(false)
	selector.SetShowPagination(false)
	selector.SetShowStatusBar(false)
	selector.SetShowTitle(false)
	selector.SetFilteringEnabled(false)
	selector.DisableQuitKeybindings()
	selector.KeyMap.NextPage = common.KeyMap.NextPage
	selector.KeyMap.PrevPage = common.KeyMap.PrevPage
	l.selector = selector
	return l
}

// SetSize implements common.Component.
func (l *Log) SetSize(width, height int) {
	l.common.SetSize(width, height)
	l.selector.SetSize(width, height)
	l.vp.SetSize(width, height)
}

// ShortHelp implements key.KeyMap.
func (l *Log) ShortHelp() []key.Binding {
	switch l.activeView {
	case logViewCommits:
		return []key.Binding{
			key.NewBinding(
				key.WithKeys(
					"l",
					"right",
				),
				key.WithHelp(
					"→",
					"select",
				),
			),
		}
	case logViewDiff:
		return []key.Binding{
			l.common.KeyMap.UpDown,
			key.NewBinding(
				key.WithKeys(
					"h",
					"left",
				),
				key.WithHelp(
					"←",
					"back",
				),
			),
		}
	default:
		return []key.Binding{}
	}
}

// Init implements tea.Model.
func (l *Log) Init() tea.Cmd {
	cmds := make([]tea.Cmd, 0)
	cmds = append(cmds, l.updateCommitsCmd)
	return tea.Batch(cmds...)
}

// Update implements tea.Model.
func (l *Log) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case RepoMsg:
		l.count = 0
		l.selector.Select(0)
		l.nextPage = 0
		l.activeView = 0
		l.repo = git.GitRepo(msg)
	case RefMsg:
		l.ref = msg
		l.count = 0
		cmds = append(cmds, l.countCommitsCmd)
	case LogCountMsg:
		l.count = int64(msg)
	case LogItemsMsg:
		cmds = append(cmds, l.selector.SetItems(msg))
		l.selector.SetPage(l.nextPage)
		l.SetSize(l.common.Width, l.common.Height)
		l.activeCommit = l.selector.SelectedItem().(LogItem).Commit
	case tea.KeyMsg, tea.MouseMsg:
		switch l.activeView {
		case logViewCommits:
			switch key := msg.(type) {
			case tea.KeyMsg:
				switch key.String() {
				case "l", "right":
					cmds = append(cmds, l.selector.SelectItem)
				}
			}
			// This is a hack for loading commits on demand based on list.Pagination.
			curPage := l.selector.Page()
			s, cmd := l.selector.Update(msg)
			m := s.(*selector.Selector)
			l.selector = m
			if m.Page() != curPage {
				l.nextPage = m.Page()
				l.selector.SetPage(curPage)
				cmds = append(cmds, l.updateCommitsCmd)
			}
			cmds = append(cmds, cmd)
		case logViewDiff:
			switch key := msg.(type) {
			case tea.KeyMsg:
				switch key.String() {
				case "h", "left":
					l.activeView = logViewCommits
				}
			}
		}
	case selector.ActiveMsg:
		switch sel := msg.IdentifiableItem.(type) {
		case LogItem:
			l.activeCommit = sel.Commit
		}
		cmds = append(cmds, updateStatusBarCmd)
	case selector.SelectMsg:
		switch sel := msg.IdentifiableItem.(type) {
		case LogItem:
			cmds = append(cmds, l.selectCommitCmd(sel.Commit))
		}
	case LogCommitMsg:
		l.selectedCommit = msg
		cmds = append(cmds, l.loadDiffCmd)
	case LogDiffMsg:
		l.currentDiff = msg
		l.vp.SetContent(
			lipgloss.JoinVertical(lipgloss.Top,
				l.renderCommit(l.selectedCommit),
				l.renderSummary(msg),
				l.renderDiff(msg),
			),
		)
		l.vp.GotoTop()
		l.activeView = logViewDiff
		cmds = append(cmds, updateStatusBarCmd)
	case tea.WindowSizeMsg:
		if l.selectedCommit != nil && l.currentDiff != nil {
			l.vp.SetContent(
				lipgloss.JoinVertical(lipgloss.Top,
					l.renderCommit(l.selectedCommit),
					l.renderSummary(l.currentDiff),
					l.renderDiff(l.currentDiff),
				),
			)
		}
		if l.repo != nil {
			cmds = append(cmds, l.updateCommitsCmd)
		}
	}
	switch l.activeView {
	case logViewDiff:
		vp, cmd := l.vp.Update(msg)
		l.vp = vp.(*viewport.Viewport)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return l, tea.Batch(cmds...)
}

// View implements tea.Model.
func (l *Log) View() string {
	switch l.activeView {
	case logViewCommits:
		return l.selector.View()
	case logViewDiff:
		return l.vp.View()
	default:
		return ""
	}
}

// StatusBarValue returns the status bar value.
func (l *Log) StatusBarValue() string {
	c := l.activeCommit
	if c == nil {
		return ""
	}
	return fmt.Sprintf("%s by %s on %s",
		c.ID.String()[:7],
		c.Author.Name,
		c.Author.When.Format("02 Jan 2006"),
	)
}

// StatusBarInfo returns the status bar info.
func (l *Log) StatusBarInfo() string {
	switch l.activeView {
	case logViewCommits:
		// We're using l.nextPage instead of l.selector.Paginator.Page because
		// of the paginator hack above.
		return fmt.Sprintf("%d/%d", l.nextPage+1, l.selector.TotalPages())
	case logViewDiff:
		return fmt.Sprintf("%.f%%", l.vp.ScrollPercent()*100)
	default:
		return ""
	}
}

func (l *Log) countCommitsCmd() tea.Msg {
	count, err := l.repo.CountCommits(l.ref)
	if err != nil {
		return common.ErrorMsg(err)
	}
	return LogCountMsg(count)
}

func (l *Log) updateCommitsCmd() tea.Msg {
	count := l.count
	if l.count == 0 {
		switch msg := l.countCommitsCmd().(type) {
		case common.ErrorMsg:
			return msg
		case LogCountMsg:
			count = int64(msg)
		}
	}
	items := make([]selector.IdentifiableItem, count)
	page := l.nextPage
	limit := l.selector.PerPage()
	skip := page * limit
	// CommitsByPage pages start at 1
	cc, err := l.repo.CommitsByPage(l.ref, page+1, limit)
	if err != nil {
		return common.ErrorMsg(err)
	}
	for i, c := range cc {
		idx := i + skip
		if int64(idx) >= count {
			break
		}
		items[idx] = LogItem{c}
	}
	return LogItemsMsg(items)
}

func (l *Log) selectCommitCmd(commit *ggit.Commit) tea.Cmd {
	return func() tea.Msg {
		return LogCommitMsg(commit)
	}
}

func (l *Log) loadDiffCmd() tea.Msg {
	diff, err := l.repo.Diff(l.selectedCommit)
	if err != nil {
		return common.ErrorMsg(err)
	}
	return LogDiffMsg(diff)
}

func styleConfig() gansi.StyleConfig {
	noColor := ""
	s := glamour.DarkStyleConfig
	s.Document.StylePrimitive.Color = &noColor
	s.CodeBlock.Chroma.Text.Color = &noColor
	s.CodeBlock.Chroma.Name.Color = &noColor
	return s
}

func renderCtx() gansi.RenderContext {
	return gansi.NewRenderContext(gansi.Options{
		ColorProfile: termenv.TrueColor,
		Styles:       styleConfig(),
	})
}

func (l *Log) renderCommit(c *ggit.Commit) string {
	s := strings.Builder{}
	// FIXME: lipgloss prints empty lines when CRLF is used
	// sanitize commit message from CRLF
	msg := strings.ReplaceAll(c.Message, "\r\n", "\n")
	s.WriteString(fmt.Sprintf("%s\n%s\n%s\n%s\n",
		l.common.Styles.LogCommitHash.Render("commit "+c.ID.String()),
		l.common.Styles.LogCommitAuthor.Render(fmt.Sprintf("Author: %s <%s>", c.Author.Name, c.Author.Email)),
		l.common.Styles.LogCommitDate.Render("Date:   "+c.Committer.When.Format(time.UnixDate)),
		l.common.Styles.LogCommitBody.Render(msg),
	))
	return wrap.String(s.String(), l.common.Width-2)
}

func (l *Log) renderSummary(diff *ggit.Diff) string {
	stats := strings.Split(diff.Stats().String(), "\n")
	for i, line := range stats {
		ch := strings.Split(line, "|")
		if len(ch) > 1 {
			adddel := ch[len(ch)-1]
			adddel = strings.ReplaceAll(adddel, "+", l.common.Styles.LogCommitStatsAdd.Render("+"))
			adddel = strings.ReplaceAll(adddel, "-", l.common.Styles.LogCommitStatsDel.Render("-"))
			stats[i] = strings.Join(ch[:len(ch)-1], "|") + "|" + adddel
		}
	}
	return wrap.String(strings.Join(stats, "\n"), l.common.Width-2)
}

func (l *Log) renderDiff(diff *ggit.Diff) string {
	var s strings.Builder
	var pr strings.Builder
	diffChroma := &gansi.CodeBlockElement{
		Code:     diff.Patch(),
		Language: "diff",
	}
	err := diffChroma.Render(&pr, renderCtx())
	if err != nil {
		s.WriteString(fmt.Sprintf("\n%s", err.Error()))
	} else {
		s.WriteString(fmt.Sprintf("\n%s", pr.String()))
	}
	return wrap.String(s.String(), l.common.Width-2)
}
