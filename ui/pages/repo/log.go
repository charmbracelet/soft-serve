package repo

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	ggit "github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/ui/common"
	"github.com/charmbracelet/soft-serve/ui/components/selector"
	"github.com/charmbracelet/soft-serve/ui/components/viewport"
	"github.com/charmbracelet/soft-serve/ui/git"
)

type view int

const (
	logView view = iota
	commitView
)

type LogCountMsg int64

type LogItemsMsg []list.Item

type Log struct {
	common     common.Common
	selector   *selector.Selector
	vp         *viewport.Viewport
	activeView view
	repo       git.GitRepo
	ref        *ggit.Reference
	count      int64
	nextPage   int
}

func NewLog(common common.Common) *Log {
	l := &Log{
		common:     common,
		vp:         viewport.New(),
		activeView: logView,
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

func (l *Log) SetSize(width, height int) {
	l.common.SetSize(width, height)
	l.selector.SetSize(width, height)
	l.vp.SetSize(width, height)
}

func (l *Log) Init() tea.Cmd {
	cmds := make([]tea.Cmd, 0)
	cmds = append(cmds, l.updateCommitsCmd)
	return tea.Batch(cmds...)
}

func (l *Log) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case RepoMsg:
		l.count = 0
		l.selector.Select(0)
		l.nextPage = 0
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
	case tea.KeyMsg, tea.MouseMsg:
		// This is a hack for loading commits on demand based on list.Pagination.
		if l.activeView == logView {
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
		}
	}
	switch l.activeView {
	case commitView:
		vp, cmd := l.vp.Update(msg)
		l.vp = vp.(*viewport.Viewport)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return l, tea.Batch(cmds...)
}

func (l *Log) View() string {
	switch l.activeView {
	case logView:
		return l.selector.View()
	case commitView:
		return l.vp.View()
	default:
		return ""
	}
}

func (l *Log) StatusBarInfo() string {
	switch l.activeView {
	case logView:
		// We're using l.nextPage instead of l.selector.Paginator.Page because
		// of the paginator hack above.
		return fmt.Sprintf("%d/%d", l.nextPage+1, l.selector.TotalPages())
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
	items := make([]list.Item, count)
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
