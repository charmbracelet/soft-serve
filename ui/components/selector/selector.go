package selector

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/soft-serve/ui/common"
)

type Selector struct {
	list   list.Model
	common *common.Common
}

func New(common *common.Common, items []list.Item) *Selector {
	l := list.New(items, ItemDelegate{common.Styles}, common.Width, common.Height)
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.DisableQuitKeybindings()
	s := &Selector{
		list:   l,
		common: common,
	}
	return s
}

func (s *Selector) SetSize(width, height int) {
	s.list.SetSize(width, height)
}

func (s *Selector) SetItems(items []list.Item) tea.Cmd {
	return s.list.SetItems(items)
}

func (s *Selector) Init() tea.Cmd {
	return nil
}

func (s *Selector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	default:
		m, cmd := s.list.Update(msg)
		s.list = m
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return s, tea.Batch(cmds...)
}

func (s *Selector) View() string {
	return s.list.View()
}
