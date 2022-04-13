package selector

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/soft-serve/ui/common"
)

type Selector struct {
	list   list.Model
	common common.Common
	active int
}

type SelectMsg string

type ActiveMsg string

func New(common common.Common, items []list.Item) *Selector {
	l := list.New(items, ItemDelegate{common.Styles}, common.Width, common.Height)
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.DisableQuitKeybindings()
	s := &Selector{
		list:   l,
		common: common,
	}
	s.SetSize(common.Width, common.Height)
	return s
}

func (s *Selector) KeyMap() list.KeyMap {
	return s.list.KeyMap
}

func (s *Selector) SetSize(width, height int) {
	s.common.SetSize(width, height)
	s.list.SetSize(width, height)
}

func (s *Selector) SetItems(items []list.Item) tea.Cmd {
	return s.list.SetItems(items)
}

func (s *Selector) Index() int {
	return s.list.Index()
}

func (s *Selector) Init() tea.Cmd {
	return s.activeCmd
}

func (s *Selector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, s.common.Keymap.Select):
			cmds = append(cmds, s.selectCmd)
		}
	}
	m, cmd := s.list.Update(msg)
	s.list = m
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	// Send ActiveMsg when index change.
	if s.active != s.list.Index() {
		cmds = append(cmds, s.activeCmd)
	}
	s.active = s.list.Index()
	return s, tea.Batch(cmds...)
}

func (s *Selector) View() string {
	return s.list.View()
}

func (s *Selector) selectCmd() tea.Msg {
	item := s.list.SelectedItem()
	i := item.(Item)
	return SelectMsg(i.Name)
}

func (s *Selector) activeCmd() tea.Msg {
	item := s.list.SelectedItem()
	i := item.(Item)
	return ActiveMsg(i.Name)
}
