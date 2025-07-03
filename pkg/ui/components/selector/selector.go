package selector

import (
	"sync"

	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/list"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/soft-serve/pkg/ui/common"
)

// Selector is a list of items that can be selected.
type Selector struct {
	*list.Model
	common      common.Common
	active      int
	filterState list.FilterState

	// XXX: we use a mutex to support concurrent access to the model. This is
	// needed to implement pagination for the Log component. list.Model does
	// not support item pagination so we hack it ourselves on top of
	// list.Model.
	mtx sync.RWMutex
}

// IdentifiableItem is an item that can be identified by a string. Implements
// list.DefaultItem.
type IdentifiableItem interface {
	list.DefaultItem
	ID() string
}

// ItemDelegate is a wrapper around list.ItemDelegate.
type ItemDelegate interface {
	list.ItemDelegate
}

// SelectMsg is a message that is sent when an item is selected.
type SelectMsg struct{ IdentifiableItem }

// ActiveMsg is a message that is sent when an item is active but not selected.
type ActiveMsg struct{ IdentifiableItem }

// New creates a new selector.
func New(common common.Common, items []IdentifiableItem, delegate ItemDelegate) *Selector {
	itms := make([]list.Item, len(items))
	for i, item := range items {
		itms[i] = item
	}
	l := list.New(itms, delegate, common.Width, common.Height)
	l.Styles.NoItems = common.Styles.NoContent
	s := &Selector{
		Model:  &l,
		common: common,
	}
	s.SetSize(common.Width, common.Height)
	return s
}

// PerPage returns the number of items per page.
func (s *Selector) PerPage() int {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	return s.Model.Paginator.PerPage
}

// SetPage sets the current page.
func (s *Selector) SetPage(page int) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.Model.Paginator.Page = page
}

// Page returns the current page.
func (s *Selector) Page() int {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	return s.Model.Paginator.Page
}

// TotalPages returns the total number of pages.
func (s *Selector) TotalPages() int {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	return s.Model.Paginator.TotalPages
}

// SetTotalPages sets the total number of pages given the number of items.
func (s *Selector) SetTotalPages(items int) int {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	return s.Model.Paginator.SetTotalPages(items)
}

// SelectedItem returns the currently selected item.
func (s *Selector) SelectedItem() IdentifiableItem {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	item := s.Model.SelectedItem()
	i, ok := item.(IdentifiableItem)
	if !ok {
		return nil
	}
	return i
}

// Select selects the item at the given index.
func (s *Selector) Select(index int) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	s.Model.Select(index)
}

// SetShowTitle sets the show title flag.
func (s *Selector) SetShowTitle(show bool) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.Model.SetShowTitle(show)
}

// SetShowHelp sets the show help flag.
func (s *Selector) SetShowHelp(show bool) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.Model.SetShowHelp(show)
}

// SetShowStatusBar sets the show status bar flag.
func (s *Selector) SetShowStatusBar(show bool) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.Model.SetShowStatusBar(show)
}

// DisableQuitKeybindings disables the quit keybindings.
func (s *Selector) DisableQuitKeybindings() {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.Model.DisableQuitKeybindings()
}

// SetShowFilter sets the show filter flag.
func (s *Selector) SetShowFilter(show bool) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.Model.SetShowFilter(show)
}

// SetShowPagination sets the show pagination flag.
func (s *Selector) SetShowPagination(show bool) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.Model.SetShowPagination(show)
}

// SetFilteringEnabled sets the filtering enabled flag.
func (s *Selector) SetFilteringEnabled(enabled bool) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.Model.SetFilteringEnabled(enabled)
}

// SetSize implements common.Component.
func (s *Selector) SetSize(width, height int) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.common.SetSize(width, height)
	s.Model.SetSize(width, height)
}

// SetItems sets the items in the selector.
func (s *Selector) SetItems(items []IdentifiableItem) tea.Cmd {
	its := make([]list.Item, len(items))
	for i, item := range items {
		its[i] = item
	}
	s.mtx.Lock()
	defer s.mtx.Unlock()
	return s.Model.SetItems(its)
}

// Index returns the index of the selected item.
func (s *Selector) Index() int {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	return s.Model.Index()
}

// Items returns the items in the selector.
func (s *Selector) Items() []list.Item {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	return s.Model.Items()
}

// VisibleItems returns all the visible items in the selector.
func (s *Selector) VisibleItems() []list.Item {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	return s.Model.VisibleItems()
}

// FilterState returns the filter state.
func (s *Selector) FilterState() list.FilterState {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	return s.Model.FilterState()
}

// CursorUp moves the cursor up.
func (s *Selector) CursorUp() {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.Model.CursorUp()
}

// CursorDown moves the cursor down.
func (s *Selector) CursorDown() {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.Model.CursorDown()
}

// Init implements tea.Model.
func (s *Selector) Init() tea.Cmd {
	return s.activeCmd
}

// Update implements tea.Model.
func (s *Selector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case tea.MouseClickMsg:
		m := msg.Mouse()
		switch m.Button {
		case tea.MouseWheelUp:
			s.CursorUp()
		case tea.MouseWheelDown:
			s.CursorDown()
		case tea.MouseLeft:
			curIdx := s.Index()
			for i, item := range s.Items() {
				item, _ := item.(IdentifiableItem)
				// Check each item to see if it's in bounds.
				if item != nil && s.common.Zone.Get(item.ID()).InBounds(msg) {
					if i == curIdx {
						cmds = append(cmds, s.SelectItemCmd)
					} else {
						s.Select(i)
					}
					break
				}
			}
		}
	case tea.KeyPressMsg:
		filterState := s.FilterState()
		switch {
		case key.Matches(msg, s.common.KeyMap.Help):
			if filterState == list.Filtering {
				return s, tea.Batch(cmds...)
			}
		case key.Matches(msg, s.common.KeyMap.Select):
			if filterState != list.Filtering {
				cmds = append(cmds, s.SelectItemCmd)
			}
		}
	case list.FilterMatchesMsg:
		cmds = append(cmds, s.activeFilterCmd)
	}
	m, cmd := s.Model.Update(msg)
	s.mtx.Lock()
	s.Model = &m
	s.mtx.Unlock()
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	// Track filter state and update active item when filter state changes.
	filterState := s.FilterState()
	if s.filterState != filterState {
		cmds = append(cmds, s.activeFilterCmd)
	}
	s.filterState = filterState
	// Send ActiveMsg when index change.
	if s.active != s.Index() {
		cmds = append(cmds, s.activeCmd)
	}
	s.active = s.Index()
	return s, tea.Batch(cmds...)
}

// View implements tea.Model.
func (s *Selector) View() string {
	return s.Model.View()
}

// SelectItemCmd is a command that selects the currently active item.
func (s *Selector) SelectItemCmd() tea.Msg {
	return SelectMsg{s.SelectedItem()}
}

func (s *Selector) activeCmd() tea.Msg {
	item := s.SelectedItem()
	return ActiveMsg{item}
}

func (s *Selector) activeFilterCmd() tea.Msg {
	// Here we use VisibleItems because when list.FilterMatchesMsg is sent,
	// VisibleItems is the only way to get the list of filtered items. The list
	// bubble should export something like list.FilterMatchesMsg.Items().
	items := s.VisibleItems()
	if len(items) == 0 {
		return nil
	}
	item := items[0]
	i, ok := item.(IdentifiableItem)
	if !ok {
		return nil
	}
	return ActiveMsg{i}
}
