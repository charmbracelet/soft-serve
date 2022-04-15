package selector

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/soft-serve/ui/common"
)

// Selector is a list of items that can be selected.
type Selector struct {
	list        list.Model
	common      common.Common
	active      int
	filterState list.FilterState
}

// IdentifiableItem is an item that can be identified by a string and extends list.Item.
type IdentifiableItem interface {
	list.DefaultItem
	ID() string
}

// SelectMsg is a message that is sent when an item is selected.
type SelectMsg string

// ActiveMsg is a message that is sent when an item is active but not selected.
type ActiveMsg string

// New creates a new selector.
func New(common common.Common, items []IdentifiableItem, delegate list.ItemDelegate) *Selector {
	itms := make([]list.Item, len(items))
	for i, item := range items {
		itms[i] = item
	}
	l := list.New(itms, delegate, common.Width, common.Height)
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

// KeyMap returns the underlying list's keymap.
func (s *Selector) KeyMap() list.KeyMap {
	return s.list.KeyMap
}

// SetSize implements common.Component.
func (s *Selector) SetSize(width, height int) {
	s.common.SetSize(width, height)
	s.list.SetSize(width, height)
}

// SetItems sets the items in the selector.
func (s *Selector) SetItems(items []list.Item) tea.Cmd {
	return s.list.SetItems(items)
}

// Index returns the index of the selected item.
func (s *Selector) Index() int {
	return s.list.Index()
}

// Init implements tea.Model.
func (s *Selector) Init() tea.Cmd {
	return s.activeCmd
}

// Update implements tea.Model.
func (s *Selector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case tea.MouseMsg:
		switch msg.Type {
		case tea.MouseWheelUp:
			s.list.CursorUp()
		case tea.MouseWheelDown:
			s.list.CursorDown()
		}
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, s.common.Keymap.Select):
			cmds = append(cmds, s.selectCmd)
		}
	case list.FilterMatchesMsg:
		cmds = append(cmds, s.activeFilterCmd)
	}
	m, cmd := s.list.Update(msg)
	s.list = m
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	// Track filter state and update active item when filter state changes.
	filterState := s.list.FilterState()
	if s.filterState != filterState {
		cmds = append(cmds, s.activeFilterCmd)
	}
	s.filterState = filterState
	// Send ActiveMsg when index change.
	if s.active != s.list.Index() {
		cmds = append(cmds, s.activeCmd)
	}
	s.active = s.list.Index()
	return s, tea.Batch(cmds...)
}

// View implements tea.Model.
func (s *Selector) View() string {
	return s.list.View()
}

func (s *Selector) selectCmd() tea.Msg {
	item := s.list.SelectedItem()
	i, ok := item.(IdentifiableItem)
	if !ok {
		return SelectMsg("")
	}
	return SelectMsg(i.ID())
}

func (s *Selector) activeCmd() tea.Msg {
	item := s.list.SelectedItem()
	i, ok := item.(IdentifiableItem)
	if !ok {
		return ActiveMsg("")
	}
	return ActiveMsg(i.ID())
}

func (s *Selector) activeFilterCmd() tea.Msg {
	// Here we use VisibleItems because when list.FilterMatchesMsg is sent,
	// VisibleItems is the only way to get the list of filtered items. The list
	// bubble should export something like list.FilterMatchesMsg.Items().
	items := s.list.VisibleItems()
	if len(items) == 0 {
		return nil
	}
	item := items[0]
	i, ok := item.(IdentifiableItem)
	if !ok {
		return nil
	}
	return ActiveMsg(i.ID())
}
