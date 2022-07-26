package repo

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/ui/common"
)

// RefItem is a git reference item.
type RefItem struct {
	*git.Reference
}

// ID implements selector.IdentifiableItem.
func (i RefItem) ID() string {
	return i.Reference.Name().String()
}

// Title implements list.DefaultItem.
func (i RefItem) Title() string {
	return i.Reference.Name().Short()
}

// Description implements list.DefaultItem.
func (i RefItem) Description() string {
	return ""
}

// Short returns the short name of the reference.
func (i RefItem) Short() string {
	return i.Reference.Name().Short()
}

// FilterValue implements list.Item.
func (i RefItem) FilterValue() string { return i.Short() }

// RefItems is a list of git references.
type RefItems []RefItem

// Len implements sort.Interface.
func (cl RefItems) Len() int { return len(cl) }

// Swap implements sort.Interface.
func (cl RefItems) Swap(i, j int) { cl[i], cl[j] = cl[j], cl[i] }

// Less implements sort.Interface.
func (cl RefItems) Less(i, j int) bool {
	return cl[i].Short() < cl[j].Short()
}

// RefItemDelegate is the delegate for the ref item.
type RefItemDelegate struct {
	common *common.Common
}

// Height implements list.ItemDelegate.
func (d RefItemDelegate) Height() int { return 1 }

// Spacing implements list.ItemDelegate.
func (d RefItemDelegate) Spacing() int { return 0 }

// Update implements list.ItemDelegate.
func (d RefItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	idx := m.Index()
	item, ok := m.SelectedItem().(RefItem)
	if !ok {
		return nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, d.common.KeyMap.Copy):
			d.common.Copy.Copy(item.Title())
			return m.SetItem(idx, item)
		}
	}
	return nil
}

// Render implements list.ItemDelegate.
func (d RefItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	s := d.common.Styles.Ref
	i, ok := listItem.(RefItem)
	if !ok {
		return
	}

	var st lipgloss.Style
	var selector string

	isTag := i.Reference.IsTag()
	isActive := index == m.Index()

	if isTag && isActive {
		st = s.Active.ItemTag
	} else if isTag {
		st = s.Normal.ItemTag
	} else if isActive {
		st = s.Active.Item
	} else {
		st = s.Normal.Item
	}

	if isActive {
		selector = s.ItemSelector.String()
	} else {
		selector = "  "
	}

	ref := i.Short()
	ref = s.ItemBranch.Render(ref)
	refMaxWidth := m.Width() -
		s.ItemSelector.GetMarginLeft() -
		s.ItemSelector.GetWidth() -
		s.Normal.Item.GetMarginLeft()
	ref = common.TruncateString(ref, refMaxWidth)
	ref = st.Render(ref)
	fmt.Fprint(w,
		d.common.Zone.Mark(
			i.ID(),
			fmt.Sprint(selector, ref),
		),
	)
}
