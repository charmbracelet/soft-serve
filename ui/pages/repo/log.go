package repo

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/soft-serve/ui/common"
	"github.com/charmbracelet/soft-serve/ui/components/selector"
	"github.com/charmbracelet/soft-serve/ui/components/viewport"
)

type view int

const (
	logView view = iota
	commitView
)

type Log struct {
	common     common.Common
	selector   *selector.Selector
	vp         *viewport.Viewport
	activeView view
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
	selector.SetShowPagination(true)
	selector.SetShowStatusBar(false)
	selector.SetShowTitle(false)
	selector.SetFilteringEnabled(false)
	selector.DisableQuitKeybindings()
	selector.KeyMap.NextPage = common.Keymap.NextPage
	selector.KeyMap.PrevPage = common.Keymap.PrevPage
	l.selector = selector
	return l
}

func (l *Log) SetSize(width, height int) {
	l.common.SetSize(width, height)
	l.selector.SetSize(width, height)
	l.vp.SetSize(width, height)
}

// func (b *Bubble) countCommits() tea.Msg {
// 	if b.ref == nil {
// 		ref, err := b.repo.HEAD()
// 		if err != nil {
// 			return common.ErrMsg{Err: err}
// 		}
// 		b.ref = ref
// 	}
// 	count, err := b.repo.CountCommits(b.ref)
// 	if err != nil {
// 		return common.ErrMsg{Err: err}
// 	}
// 	return countMsg(count)
// }

// func (b *Bubble) updateItems() tea.Msg {
// 	if b.count == 0 {
// 		b.count = int64(b.countCommits().(countMsg))
// 	}
// 	count := b.count
// 	items := make([]list.Item, count)
// 	page := b.nextPage
// 	limit := b.list.Paginator.PerPage
// 	skip := page * limit
// 	// CommitsByPage pages start at 1
// 	cc, err := b.repo.CommitsByPage(b.ref, page+1, limit)
// 	if err != nil {
// 		return common.ErrMsg{Err: err}
// 	}
// 	for i, c := range cc {
// 		idx := i + skip
// 		if int64(idx) >= count {
// 			break
// 		}
// 		items[idx] = item{c}
// 	}
// 	b.list.SetItems(items)
// 	b.SetSize(b.width, b.height)
// 	return itemsMsg{}
// }

func (l *Log) Init() tea.Cmd {
	return nil
}

func (l *Log) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return l, nil
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
