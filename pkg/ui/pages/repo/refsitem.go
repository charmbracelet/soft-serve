package repo

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/list"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/pkg/ui/common"
	"github.com/dustin/go-humanize"
	"github.com/muesli/reflow/truncate"
)

// RefItem is a git reference item.
type RefItem struct {
	*git.Reference
	*git.Tag
	*git.Commit
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
	if cl[i].Commit != nil && cl[j].Commit != nil {
		return cl[i].Author.When.After(cl[j].Author.When)
	} else if cl[i].Commit != nil && cl[j].Commit == nil {
		return true
	}
	return false
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
	item, ok := m.SelectedItem().(RefItem)
	if !ok {
		return nil
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, d.common.KeyMap.Copy):
			return copyCmd(item.ID(), fmt.Sprintf("Reference %q copied to clipboard", item.ID()))
		}
	}
	return nil
}

// Render implements list.ItemDelegate.
func (d RefItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(RefItem)
	if !ok {
		return
	}

	isTag := i.IsTag()
	isActive := index == m.Index()
	s := d.common.Styles.Ref
	st := s.Normal
	selector := "  "
	if isActive {
		st = s.Active
		selector = s.ItemSelector.String()
	}

	horizontalFrameSize := st.Base.GetHorizontalFrameSize()
	var itemSt lipgloss.Style
	if isTag && isActive {
		itemSt = st.ItemTag
	} else if isTag {
		itemSt = st.ItemTag
	} else if isActive {
		itemSt = st.Item
	} else {
		itemSt = st.Item
	}

	var sha string
	c := i.Commit
	if c != nil {
		sha = c.ID.String()[:7]
	}

	ref := i.Short()

	var desc string
	//nolint:nestif // Complex UI logic requires nested conditions
	if isTag {
		if c != nil {
			date := c.Committer.When.Format("Jan 02")
			if c.Committer.When.Year() != time.Now().Year() {
				date += fmt.Sprintf(" %d", c.Committer.When.Year())
			}
			desc += " " + st.ItemDesc.Render(date)
		}

		t := i.Tag
		if t != nil {
			msgSt := st.ItemDesc.Faint(false)
			msg := t.Message()
			nl := strings.Index(msg, "\n")
			if nl > 0 {
				msg = msg[:nl]
			}
			msg = strings.TrimSpace(msg)
			if msg != "" {
				msgMargin := m.Width() -
					horizontalFrameSize -
					lipgloss.Width(selector) -
					lipgloss.Width(ref) -
					lipgloss.Width(desc) -
					lipgloss.Width(sha) -
					3 // 3 is for the paddings and truncation symbol
				if msgMargin >= 0 {
					msg = common.TruncateString(msg, msgMargin)
					desc = " " + msgSt.Render(msg) + desc
				}
			}
		}
	} else if c != nil {
		onMargin := m.Width() -
			horizontalFrameSize -
			lipgloss.Width(selector) -
			lipgloss.Width(ref) -
			lipgloss.Width(desc) -
			lipgloss.Width(sha) -
			2 // 2 is for the padding and truncation symbol
		if onMargin >= 0 {
			on := common.TruncateString("updated "+humanize.Time(c.Committer.When), onMargin)
			desc += " " + st.ItemDesc.Render(on)
		}
	}

	var hash string
	ref = itemSt.Render(ref)
	hashMargin := m.Width() -
		horizontalFrameSize -
		lipgloss.Width(selector) -
		lipgloss.Width(ref) -
		lipgloss.Width(desc) -
		lipgloss.Width(sha) -
		1 // 1 is for the left padding
	if hashMargin >= 0 {
		hash = strings.Repeat(" ", hashMargin) + st.ItemHash.
			Align(lipgloss.Right).
			PaddingLeft(1).
			Render(sha)
	}
	fmt.Fprint(w, //nolint:errcheck
		d.common.Zone.Mark(
			i.ID(),
			st.Base.Render(
				lipgloss.JoinHorizontal(lipgloss.Top,
					truncate.String(selector+ref+desc+hash,
						uint(m.Width()-horizontalFrameSize)), //nolint:gosec
				),
			),
		),
	)
}
