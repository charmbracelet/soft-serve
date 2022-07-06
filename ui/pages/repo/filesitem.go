package repo

import (
	"fmt"
	"io"
	"io/fs"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/ui/common"
	"github.com/dustin/go-humanize"
)

// FileItem is a list item for a file.
type FileItem struct {
	entry *git.TreeEntry
}

// ID returns the ID of the file item.
func (i FileItem) ID() string {
	return i.entry.Name()
}

// Title returns the title of the file item.
func (i FileItem) Title() string {
	return i.entry.Name()
}

// Description returns the description of the file item.
func (i FileItem) Description() string {
	return ""
}

// Mode returns the mode of the file item.
func (i FileItem) Mode() fs.FileMode {
	return i.entry.Mode()
}

// FilterValue implements list.Item.
func (i FileItem) FilterValue() string { return i.Title() }

// FileItems is a list of file items.
type FileItems []FileItem

// Len implements sort.Interface.
func (cl FileItems) Len() int { return len(cl) }

// Swap implements sort.Interface.
func (cl FileItems) Swap(i, j int) { cl[i], cl[j] = cl[j], cl[i] }

// Less implements sort.Interface.
func (cl FileItems) Less(i, j int) bool {
	if cl[i].entry.IsTree() && cl[j].entry.IsTree() {
		return cl[i].Title() < cl[j].Title()
	} else if cl[i].entry.IsTree() {
		return true
	} else if cl[j].entry.IsTree() {
		return false
	} else {
		return cl[i].Title() < cl[j].Title()
	}
}

// FileItemDelegate is the delegate for the file item list.
type FileItemDelegate struct {
	common *common.Common
}

// Height returns the height of the file item list. Implements list.ItemDelegate.
func (d FileItemDelegate) Height() int { return 1 }

// Spacing returns the spacing of the file item list. Implements list.ItemDelegate.
func (d FileItemDelegate) Spacing() int { return 0 }

// Update implements list.ItemDelegate.
func (d FileItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	idx := m.Index()
	item, ok := m.SelectedItem().(FileItem)
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
func (d FileItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	s := d.common.Styles.Tree
	i, ok := listItem.(FileItem)
	if !ok {
		return
	}

	name := i.Title()
	size := humanize.Bytes(uint64(i.entry.Size()))
	size = strings.ReplaceAll(size, " ", "")
	sizeLen := lipgloss.Width(size)
	if i.entry.IsTree() {
		size = strings.Repeat(" ", sizeLen)
		if index == m.Index() {
			name = s.FileDirActive.Render(name)
		} else {
			name = s.FileDirInactive.Render(name)
		}
	}
	var nameStyle, sizeStyle, modeStyle lipgloss.Style
	mode := i.Mode()
	if index == m.Index() {
		nameStyle = s.ItemActive
		sizeStyle = s.FileSizeActive
		modeStyle = s.FileModeActive
		fmt.Fprint(w, s.ItemSelector.Render(">"))
	} else {
		nameStyle = s.ItemInactive
		sizeStyle = s.FileSizeInactive
		modeStyle = s.FileModeInactive
		fmt.Fprint(w, s.ItemSelector.Render(" "))
	}
	sizeStyle = sizeStyle.Copy().
		Width(8).
		Align(lipgloss.Right).
		MarginLeft(1)
	leftMargin := s.ItemSelector.GetMarginLeft() +
		s.ItemSelector.GetWidth() +
		s.FileModeInactive.GetMarginLeft() +
		s.FileModeInactive.GetWidth() +
		nameStyle.GetMarginLeft() +
		sizeStyle.GetHorizontalFrameSize()
	name = common.TruncateString(name, m.Width()-leftMargin)
	name = nameStyle.Render(name)
	size = sizeStyle.Render(size)
	modeStr := modeStyle.Render(mode.String())
	truncate := lipgloss.NewStyle().MaxWidth(m.Width() -
		s.ItemSelector.GetHorizontalFrameSize() -
		s.ItemSelector.GetWidth())
	fmt.Fprint(w,
		truncate.Render(fmt.Sprintf("%s%s%s",
			modeStr,
			size,
			name,
		)),
	)
}
