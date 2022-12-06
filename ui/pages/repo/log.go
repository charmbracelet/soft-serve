package repo

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	gansi "github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/lipgloss"
	ggit "github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/ui/common"
	"github.com/charmbracelet/soft-serve/ui/components/footer"
	"github.com/charmbracelet/soft-serve/ui/components/selector"
	"github.com/charmbracelet/soft-serve/ui/components/viewport"
	"github.com/charmbracelet/soft-serve/ui/git"
	"github.com/muesli/reflow/wrap"
	"github.com/muesli/termenv"
)

var waitBeforeLoading = time.Millisecond * 100

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
	repo           *git.Repository
	ref            *ggit.Reference
	count          int64
	nextPage       int
	activeCommit   *ggit.Commit
	selectedCommit *ggit.Commit
	currentDiff    *ggit.Diff
	loadingTime    time.Time
	loading        bool
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
		return []key.Binding{
			l.common.KeyMap.UpDown,
			l.common.KeyMap.BackItem,
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
		k := l.vp.KeyMap
		b = append(b, []key.Binding{
			l.common.KeyMap.BackItem,
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
			},
		}...)
	}
	return b
}

func (l *Log) startLoading() tea.Cmd {
	l.loadingTime = time.Now()
	l.loading = true
	return l.spinner.Tick
}

func (l *Log) stopLoading() tea.Cmd {
	l.loading = false
	return updateStatusBarCmd
}

// Init implements tea.Model.
func (l *Log) Init() tea.Cmd {
	l.activeView = logViewCommits
	l.nextPage = 0
	l.count = 0
	l.activeCommit = nil
	l.selectedCommit = nil
	l.selector.Select(0)
	return tea.Batch(
		l.updateCommitsCmd,
		// start loading on init
		l.startLoading(),
	)
}

// Update implements tea.Model.
func (l *Log) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case RepoMsg:
		l.repo = msg
	case RefMsg:
		l.ref = msg
		cmds = append(cmds, l.Init())
	case LogCountMsg:
		l.count = int64(msg)
	case LogItemsMsg:
		cmds = append(cmds,
			l.selector.SetItems(msg),
			// stop loading after receiving items
			l.stopLoading(),
		)
		l.selector.SetPage(l.nextPage)
		l.SetSize(l.common.Width, l.common.Height)
		i := l.selector.SelectedItem()
		if i != nil {
			l.activeCommit = i.(LogItem).Commit
		}
	case tea.KeyMsg, tea.MouseMsg:
		switch l.activeView {
		case logViewCommits:
			switch kmsg := msg.(type) {
			case tea.KeyMsg:
				switch {
				case key.Matches(kmsg, l.common.KeyMap.SelectItem):
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
				cmds = append(cmds,
					l.updateCommitsCmd,
					l.startLoading(),
				)
			}
			cmds = append(cmds, cmd)
		case logViewDiff:
			switch kmsg := msg.(type) {
			case tea.KeyMsg:
				switch {
				case key.Matches(kmsg, l.common.KeyMap.BackItem):
					cmds = append(cmds, backCmd)
				}
			}
		}
	case BackMsg:
		if l.activeView == logViewDiff {
			l.activeView = logViewCommits
			l.selectedCommit = nil
			cmds = append(cmds, updateStatusBarCmd)
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
			lipgloss.JoinVertical(lipgloss.Top,
				l.renderCommit(l.selectedCommit),
				l.renderSummary(msg),
				l.renderDiff(msg),
			),
		)
		l.vp.GotoTop()
		l.activeView = logViewDiff
		cmds = append(cmds,
			updateStatusBarCmd,
			// stop loading after setting the viewport content
			l.stopLoading(),
		)
	case footer.ToggleFooterMsg:
		cmds = append(cmds, l.updateCommitsCmd)
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
			cmds = append(cmds,
				l.updateCommitsCmd,
				// start loading on resize since the number of commits per page
				// might change and we'd need to load more commits.
				l.startLoading(),
			)
		}
	}
	if l.loading {
		s, cmd := l.spinner.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		l.spinner = s
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
	if l.loading && l.loadingTime.Add(waitBeforeLoading).Before(time.Now()) {
		msg := fmt.Sprintf("%s loading commit", l.spinner.View())
		if l.selectedCommit == nil {
			msg += "s"
		}
		msg += "…"
		return l.common.Styles.SpinnerContainer.Copy().
			Height(l.common.Height).
			Render(msg)
	}
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
	if l.loading {
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
	value := c.ID.String()
	if who != "" {
		value += " by " + who
	}
	return value
}

// StatusBarInfo returns the status bar info.
func (l *Log) StatusBarInfo() string {
	switch l.activeView {
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

func (l *Log) countCommitsCmd() tea.Msg {
	if l.ref == nil {
		log.Printf("ui: log: ref is nil")
		return common.ErrorMsg(errNoRef)
	}
	count, err := l.repo.Repo.Repository().CountCommits(l.ref)
	if err != nil {
		log.Printf("ui: error counting commits: %v", err)
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
	if l.ref == nil {
		log.Printf("ui: log: ref is nil")
		return common.ErrorMsg(errNoRef)
	}
	items := make([]selector.IdentifiableItem, count)
	page := l.nextPage
	limit := l.selector.PerPage()
	skip := page * limit
	// CommitsByPage pages start at 1
	cc, err := l.repo.Repo.Repository().CommitsByPage(l.ref, page+1, limit)
	if err != nil {
		log.Printf("ui: error loading commits: %v", err)
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

func (l *Log) selectCommitCmd(commit *ggit.Commit) tea.Cmd {
	return func() tea.Msg {
		return LogCommitMsg(commit)
	}
}

func (l *Log) loadDiffCmd() tea.Msg {
	diff, err := l.repo.Repo.Repository().Diff(l.selectedCommit)
	if err != nil {
		log.Printf("ui: error loading diff: %v", err)
		return common.ErrorMsg(err)
	}
	return LogDiffMsg(diff)
}

func renderCtx() gansi.RenderContext {
	return gansi.NewRenderContext(gansi.Options{
		ColorProfile: termenv.TrueColor,
		Styles:       common.StyleConfig(),
	})
}

func (l *Log) renderCommit(c *ggit.Commit) string {
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

func (l *Log) renderSummary(diff *ggit.Diff) string {
	stats := strings.Split(diff.Stats().String(), "\n")
	for i, line := range stats {
		ch := strings.Split(line, "|")
		if len(ch) > 1 {
			adddel := ch[len(ch)-1]
			adddel = strings.ReplaceAll(adddel, "+", l.common.Styles.Log.CommitStatsAdd.Render("+"))
			adddel = strings.ReplaceAll(adddel, "-", l.common.Styles.Log.CommitStatsDel.Render("-"))
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
	return wrap.String(s.String(), l.common.Width)
}
