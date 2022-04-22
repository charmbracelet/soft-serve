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
		selector:   selector.New(common, []selector.IdentifiableItem{}, LogItemDelegate{common.Styles}),
		vp:         viewport.New(),
		activeView: logView,
	}
	return l
}

func (l *Log) SetSize(width, height int) {
	l.common.SetSize(width, height)
	l.selector.SetSize(width, height)
	l.vp.SetSize(width, height)
}

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
