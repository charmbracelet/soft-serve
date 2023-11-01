package repo

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/soft-serve/pkg/ui/common"
	gitm "github.com/aymanbagabas/git-module"
)

// StashItem represents a stash item.
type StashItem struct{ *gitm.Stash }

// ID returns the ID of the stash item.
func (i StashItem) ID() string {
	return fmt.Sprintf("stash@{%d}", i.Index)
}

// Title returns the title of the stash item.
func (i StashItem) Title() string {
	return i.Message
}

// Description returns the description of the stash item.
func (i StashItem) Description() string {
	return ""
}

// FilterValue implements list.Item.
func (i StashItem) FilterValue() string { return i.Title() }

// StashItems is a list of stash items.
type StashItems []StashItem

// Len implements sort.Interface.
func (cl StashItems) Len() int { return len(cl) }

// Swap implements sort.Interface.
func (cl StashItems) Swap(i, j int) { cl[i], cl[j] = cl[j], cl[i] }

// Less implements sort.Interface.
func (cl StashItems) Less(i, j int) bool {
	return cl[i].Index < cl[j].Index
}

// StashItemDelegate is a delegate for stash items.
type StashItemDelegate struct {
	common *common.Common
}

// Height returns the height of the stash item list. Implements list.ItemDelegate.
func (d StashItemDelegate) Height() int { return 1 }

// Spacing implements list.ItemDelegate.
func (d StashItemDelegate) Spacing() int { return 0 }

// Update implements list.ItemDelegate.
func (d StashItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	item, ok := m.SelectedItem().(StashItem)
	if !ok {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, d.common.KeyMap.Copy):
			return copyCmd(item.Title(), fmt.Sprintf("Stash message %q copied to clipboard", item.Title()))
		}
	}

	return nil
}

// Render implements list.ItemDelegate.
func (d StashItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(StashItem)
	if !ok {
		return
	}

	s := d.common.Styles.Stash

	st := s.Normal.Message
	selector := " "
	if index == m.Index() {
		selector = "> "
		st = s.Active.Message
	}

	selector = s.Selector.Render(selector)
	title := st.Render(item.Title())
	fmt.Fprint(w, d.common.Zone.Mark(
		item.ID(),
		common.TruncateString(fmt.Sprintf("%s%s",
			selector,
			title,
		), m.Width()-
			s.Selector.GetWidth()-
			st.GetHorizontalFrameSize(),
		),
	))
}
