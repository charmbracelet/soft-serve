package repo

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	gansi "charm.land/glamour/v2/ansi"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/ui/common"
	"github.com/charmbracelet/soft-serve/pkg/ui/components/footer"
	"github.com/charmbracelet/soft-serve/pkg/ui/components/selector"
	"github.com/charmbracelet/soft-serve/pkg/ui/components/viewport"
	"github.com/charmbracelet/soft-serve/pkg/ui/styles"
	"github.com/muesli/reflow/wrap"
)

var waitBeforeLoading = time.Millisecond * 100

type logView int

const (
	logViewLoading logView = iota
	logViewCommits
	logViewDiff
)

// LogCountMsg is a message that contains the number of commits in a repo.
type LogCountMsg int64

// LogItemsMsg is a message that contains a slice of LogItem.
type LogItemsMsg []selector.IdentifiableItem

// LogCommitMsg is a message that contains a git commit.
type LogCommitMsg *git.Commit

// LogDiffMsg is a message that contains a git diff.
type LogDiffMsg *git.Diff

// Log is a model that displays a list of commits and their diffs.
type Log struct {
	common         common.Common
	selector       *selector.Selector
	vp             *viewport.Viewport
	activeView     logView
	repo           proto.Repository
	ref            *git.Reference
	count          int64
	nextPage       int
	activeCommit   *git.Commit
	selectedCommit *git.Commit
	currentDiff    *git.Diff
	loadingTime    time.Time
	spinner        spinner.Model
}

// NewLog creates a new Log model.
func NewLog(common common.Common) *Log {
	l := &Log{
		common:     common,
		vp:         viewport.New(common),
		activeView: logViewCommits,
	}
	selector := selector.New(common, []selector.IdentifiableItem{}, LogItemDelegate{&common})
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
	s := spinner.New(spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(common.Styles.Spinner))
	l.spinner = s
	return l
}

// Path implements common.TabComponent.
func (l *Log) Path() string {
	switch l.activeView {
	case logViewCommits:
		return ""
	default:
		return "diff" // XXX: this is a place holder and doesn't mean anything
	}
}

// TabName returns the name of the tab.
func (l *Log) TabName() string {
	return "Commits"
}

// SetSize implements common.Component.
func (l *Log) SetSize(width, height int) {
	l.common.SetSize(width, height)
	l.selector.SetSize(width, height)
	l.vp.SetSize(width, height)
}

// ShortHelp implements help.KeyMap.
func (l *Log) ShortHelp() []key.Binding {
	switch l.activeView {
	case logViewCommits:
		copyKey := l.common.KeyMap.Copy
		copyKey.SetHelp("c", "copy hash")
		return []key.Binding{
			l.common.KeyMap.UpDown,
			l.common.KeyMap.SelectItem,
			copyKey,
		}
	case logViewDiff:
		copyKey := l.common.KeyMap.Copy
		copyKey.SetHelp("c", "copy diff")
		return []key.Binding{
			l.common.KeyMap.UpDown,
			l.common.KeyMap.BackItem,
			copyKey,
			l.common.KeyMap.GotoTop,
			l.common.KeyMap.GotoBottom,
		}
	default:
		return []key.Binding{}
	}
}

// FullHelp implements help.KeyMap.
func (l *Log) FullHelp() [][]key.Binding {
	k := l.selector.KeyMap
	b := make([][]key.Binding, 0)
	switch l.activeView {
	case logViewCommits:
		copyKey := l.common.KeyMap.Copy
		copyKey.SetHelp("c", "copy hash")
		b = append(b, []key.Binding{
			l.common.KeyMap.SelectItem,
			l.common.KeyMap.BackItem,
		})
		b = append(b, [][]key.Binding{
			{
				copyKey,
				k.CursorUp,
				k.CursorDown,
			},
			{
				k.NextPage,
				k.PrevPage,
				k.GoToStart,
				k.GoToEnd,
			},
		}...)
	case logViewDiff:
		copyKey := l.common.KeyMap.Copy
		copyKey.SetHelp("c", "copy diff")
		k := l.vp.KeyMap
		b = append(b, []key.Binding{
			l.common.KeyMap.BackItem,
			copyKey,
		})
		b = append(b, [][]key.Binding{
			{
				k.PageDown,
				k.PageUp,
				k.HalfPageDown,
				k.HalfPageUp,
			},
			{
				k.Down,
				k.Up,
				l.common.KeyMap.GotoTop,
				l.common.KeyMap.GotoBottom,
			},
		}...)
	}
	return b
}

func (l *Log) startLoading() tea.Cmd {
	l.loadingTime = time.Now()
	l.activeView = logViewLoading
	return l.spinner.Tick
}

// Init implements tea.Model.
func (l *Log) Init() tea.Cmd {
	l.activeView = logViewCommits
	l.nextPage = 0
	l.count = 0
	l.activeCommit = nil
	l.selectedCommit = nil
	return tea.Batch(
		l.countCommitsCmd,
		// start loading on init
		l.startLoading(),
	)
}

// Update implements tea.Model.
func (l *Log) Update(msg tea.Msg) (common.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case RepoMsg:
		l.repo = msg
	case RefMsg:
		l.ref = msg
		l.selector.Select(0)
		cmds = append(cmds, l.Init())
	case LogCountMsg:
		l.count = int64(msg)
		l.selector.SetTotalPages(int(msg))
		l.selector.SetItems(make([]selector.IdentifiableItem, l.count))
		cmds = append(cmds, l.updateCommitsCmd)
	case LogItemsMsg:
		// stop loading after receiving items
		l.activeView = logViewCommits
		cmds = append(cmds, l.selector.SetItems(msg))
		l.selector.SetPage(l.nextPage)
		l.SetSize(l.common.Width, l.common.Height)
		i := l.selector.SelectedItem()
		if i != nil {
			l.activeCommit = i.(LogItem).Commit
		}
	case tea.KeyPressMsg, tea.MouseClickMsg:
		switch l.activeView {
		case logViewCommits:
			switch kmsg := msg.(type) {
			case tea.KeyPressMsg:
				switch {
				case key.Matches(kmsg, l.common.KeyMap.SelectItem):
					cmds = append(cmds, l.selector.SelectItemCmd)
				}
			}
			// XXX: This is a hack for loading commits on demand based on
			// list.Pagination.
			curPage := l.selector.Page()
			s, cmd := l.selector.Update(msg)
			m := s.(*selector.Selector)
			l.selector = m
			if m.Page() != curPage {
				l.nextPage = m.Page()
				l.selector.SetPage(curPage)
				cmds = append(cmds,
					l.updateCommitsCmd,
					l.startLoading(),
				)
			}
			cmds = append(cmds, cmd)
		case logViewDiff:
			switch kmsg := msg.(type) {
			case tea.KeyPressMsg:
				switch {
				case key.Matches(kmsg, l.common.KeyMap.BackItem):
					l.goBack()
				case key.Matches(kmsg, l.common.KeyMap.Copy):
					if l.currentDiff != nil {
						cmds = append(cmds, copyCmd(l.currentDiff.Patch(), "Commit diff copied to clipboard"))
					}
				}
			}
		}
	case GoBackMsg:
		l.goBack()
	case selector.ActiveMsg:
		switch sel := msg.IdentifiableItem.(type) {
		case LogItem:
			l.activeCommit = sel.Commit
		}
	case selector.SelectMsg:
		switch sel := msg.IdentifiableItem.(type) {
		case LogItem:
			cmds = append(cmds,
				l.selectCommitCmd(sel.Commit),
				l.startLoading(),
			)
		}
	case LogCommitMsg:
		l.selectedCommit = msg
		cmds = append(cmds, l.loadDiffCmd)
	case LogDiffMsg:
		l.currentDiff = msg
		l.vp.SetContent(
			lipgloss.JoinVertical(lipgloss.Left,
				l.renderCommit(l.selectedCommit),
				renderSummary(msg, l.common.Styles, l.common.Width),
				renderDiff(msg, l.common.Width),
			),
		)
		l.vp.GotoTop()
		l.activeView = logViewDiff
	case footer.ToggleFooterMsg:
		cmds = append(cmds, l.updateCommitsCmd)
	case tea.WindowSizeMsg:
		l.SetSize(msg.Width, msg.Height)
		if l.selectedCommit != nil && l.currentDiff != nil {
			l.vp.SetContent(
				lipgloss.JoinVertical(lipgloss.Left,
					l.renderCommit(l.selectedCommit),
					renderSummary(l.currentDiff, l.common.Styles, l.common.Width),
					renderDiff(l.currentDiff, l.common.Width),
				),
			)
		}
		if l.repo != nil && l.ref != nil {
			cmds = append(cmds,
				l.updateCommitsCmd,
				// start loading on resize since the number of commits per page
				// might change and we'd need to load more commits.
				l.startLoading(),
			)
		}
	case EmptyRepoMsg:
		l.ref = nil
		l.activeView = logViewCommits
		l.nextPage = 0
		l.count = 0
		l.activeCommit = nil
		l.selectedCommit = nil
		l.selector.Select(0)
		cmds = append(cmds,
			l.setItems([]selector.IdentifiableItem{}),
		)
	case spinner.TickMsg:
		if l.activeView == logViewLoading && l.spinner.ID() == msg.ID {
			s, cmd := l.spinner.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			l.spinner = s
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
	case logViewLoading:
		if l.loadingTime.Add(waitBeforeLoading).Before(time.Now()) {
			msg := fmt.Sprintf("%s loading commit", l.spinner.View())
			if l.selectedCommit == nil {
				msg += "s"
			}
			msg += "…"
			return l.common.Styles.SpinnerContainer.
				Height(l.common.Height).
				Render(msg)
		}
		fallthrough
	case logViewCommits:
		return l.selector.View()
	case logViewDiff:
		return l.vp.View()
	default:
		return ""
	}
}

// SpinnerID implements common.TabComponent.
func (l *Log) SpinnerID() int {
	return l.spinner.ID()
}

// StatusBarValue returns the status bar value.
func (l *Log) StatusBarValue() string {
	if l.activeView == logViewLoading {
		return ""
	}
	c := l.activeCommit
	if c == nil {
		return ""
	}
	who := c.Author.Name
	if email := c.Author.Email; email != "" {
		who += " <" + email + ">"
	}
	value := c.ID.String()[:7]
	if who != "" {
		value += " by " + who
	}
	return value
}

// StatusBarInfo returns the status bar info.
func (l *Log) StatusBarInfo() string {
	switch l.activeView {
	case logViewLoading:
		if l.count == 0 {
			return ""
		}
		fallthrough
	case logViewCommits:
		// We're using l.nextPage instead of l.selector.Paginator.Page because
		// of the paginator hack above.
		return fmt.Sprintf("p. %d/%d", l.nextPage+1, l.selector.TotalPages())
	case logViewDiff:
		return fmt.Sprintf("☰ %.f%%", l.vp.ScrollPercent()*100)
	default:
		return ""
	}
}

func (l *Log) goBack() {
	if l.activeView == logViewDiff {
		l.activeView = logViewCommits
		l.selectedCommit = nil
	}
}

func (l *Log) countCommitsCmd() tea.Msg {
	if l.ref == nil {
		return nil
	}
	r, err := l.repo.Open()
	if err != nil {
		return common.ErrorMsg(err)
	}
	count, err := r.CountCommits(l.ref)
	if err != nil {
		l.common.Logger.Debugf("ui: error counting commits: %v", err)
		return common.ErrorMsg(err)
	}
	return LogCountMsg(count)
}

func (l *Log) updateCommitsCmd() tea.Msg {
	if l.ref == nil {
		return nil
	}
	r, err := l.repo.Open()
	if err != nil {
		return common.ErrorMsg(err)
	}

	count := l.count
	if count == 0 {
		return LogItemsMsg([]selector.IdentifiableItem{})
	}

	page := l.nextPage
	limit := l.selector.PerPage()
	skip := page * limit
	ref := l.ref
	items := make([]selector.IdentifiableItem, count)
	// CommitsByPage pages start at 1
	cc, err := r.CommitsByPage(ref, page+1, limit)
	if err != nil {
		l.common.Logger.Debugf("ui: error loading commits: %v", err)
		return common.ErrorMsg(err)
	}
	for i, c := range cc {
		idx := i + skip
		if int64(idx) >= count {
			break
		}
		items[idx] = LogItem{Commit: c}
	}
	return LogItemsMsg(items)
}

func (l *Log) selectCommitCmd(commit *git.Commit) tea.Cmd {
	return func() tea.Msg {
		return LogCommitMsg(commit)
	}
}

func (l *Log) loadDiffCmd() tea.Msg {
	if l.selectedCommit == nil {
		return nil
	}
	r, err := l.repo.Open()
	if err != nil {
		l.common.Logger.Debugf("ui: error loading diff repository: %v", err)
		return common.ErrorMsg(err)
	}
	diff, err := r.Diff(l.selectedCommit)
	if err != nil {
		l.common.Logger.Debugf("ui: error loading diff: %v", err)
		return common.ErrorMsg(err)
	}
	return LogDiffMsg(diff)
}

func (l *Log) renderCommit(c *git.Commit) string {
	s := strings.Builder{}
	// FIXME: lipgloss prints empty lines when CRLF is used
	// sanitize commit message from CRLF
	msg := strings.ReplaceAll(c.Message, "\r\n", "\n")
	s.WriteString(fmt.Sprintf("%s\n%s\n%s\n%s\n",
		l.common.Styles.Log.CommitHash.Render("commit "+c.ID.String()),
		l.common.Styles.Log.CommitAuthor.Render(fmt.Sprintf("Author: %s <%s>", c.Author.Name, c.Author.Email)),
		l.common.Styles.Log.CommitDate.Render("Date:   "+c.Committer.When.Format(time.UnixDate)),
		l.common.Styles.Log.CommitBody.Render(msg),
	))
	return wrap.String(s.String(), l.common.Width-2)
}

func renderSummary(diff *git.Diff, styles *styles.Styles, width int) string {
	stats := strings.Split(diff.Stats().String(), "\n")
	for i, line := range stats {
		ch := strings.Split(line, "|")
		if len(ch) > 1 {
			adddel := ch[len(ch)-1]
			adddel = strings.ReplaceAll(adddel, "+", styles.Log.CommitStatsAdd.Render("+"))
			adddel = strings.ReplaceAll(adddel, "-", styles.Log.CommitStatsDel.Render("-"))
			stats[i] = strings.Join(ch[:len(ch)-1], "|") + "|" + adddel
		}
	}
	return wrap.String(strings.Join(stats, "\n"), width-2)
}

func renderDiff(diff *git.Diff, width int) string {
	var s strings.Builder
	var pr strings.Builder
	diffChroma := &gansi.CodeBlockElement{
		Code:     diff.Patch(),
		Language: "diff",
	}
	err := diffChroma.Render(&pr, common.StyleRenderer())
	if err != nil {
		s.WriteString(fmt.Sprintf("\n%s", err.Error()))
	} else {
		s.WriteString(fmt.Sprintf("\n%s", pr.String()))
	}
	return wrap.String(s.String(), width)
}

func (l *Log) setItems(items []selector.IdentifiableItem) tea.Cmd {
	return func() tea.Msg {
		return LogItemsMsg(items)
	}
}
