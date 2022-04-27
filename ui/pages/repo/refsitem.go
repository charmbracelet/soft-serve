package repo

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/tui/common"
	"github.com/charmbracelet/soft-serve/ui/styles"
)

type RefItem struct {
	*git.Reference
}

func (i RefItem) ID() string {
	return i.Reference.Name().String()
}

func (i RefItem) Title() string {
	return i.Reference.Name().Short()
}

func (i RefItem) Description() string {
	return ""
}

func (i RefItem) Short() string {
	return i.Reference.Name().Short()
}

func (i RefItem) FilterValue() string { return i.Short() }

type RefItems []RefItem

func (cl RefItems) Len() int      { return len(cl) }
func (cl RefItems) Swap(i, j int) { cl[i], cl[j] = cl[j], cl[i] }
func (cl RefItems) Less(i, j int) bool {
	return cl[i].Short() < cl[j].Short()
}

type RefItemDelegate struct {
	style *styles.Styles
}

func (d RefItemDelegate) Height() int                               { return 1 }
func (d RefItemDelegate) Spacing() int                              { return 0 }
func (d RefItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d RefItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	s := d.style
	i, ok := listItem.(RefItem)
	if !ok {
		return
	}

	ref := i.Short()
	if i.Reference.IsTag() {
		ref = s.RefItemTag.Render(ref)
	}
	ref = s.RefItemBranch.Render(ref)
	refMaxWidth := m.Width() -
		s.RefItemSelector.GetMarginLeft() -
		s.RefItemSelector.GetWidth() -
		s.RefItemInactive.GetMarginLeft()
	ref = common.TruncateString(ref, refMaxWidth, "…")
	if index == m.Index() {
		fmt.Fprint(w, s.RefItemSelector.Render(">")+
			s.RefItemActive.Render(ref))
	} else {
		fmt.Fprint(w, s.LogItemSelector.Render(" ")+
			s.RefItemInactive.Render(ref))
	}
}
