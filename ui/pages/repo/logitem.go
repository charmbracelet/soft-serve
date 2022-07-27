package repo

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/ui/common"
	"github.com/muesli/reflow/truncate"
)

// LogItem is a item in the log list that displays a git commit.
type LogItem struct {
	*git.Commit
	copied time.Time
}

// ID implements selector.IdentifiableItem.
func (i LogItem) ID() string {
	return i.Hash()
}

func (i LogItem) Hash() string {
	return i.Commit.ID.String()
}

// Title returns the item title. Implements list.DefaultItem.
func (i LogItem) Title() string {
	if i.Commit != nil {
		return strings.Split(i.Commit.Message, "\n")[0]
	}
	return ""
}

// Description returns the item description. Implements list.DefaultItem.
func (i LogItem) Description() string { return "" }

// FilterValue implements list.Item.
func (i LogItem) FilterValue() string { return i.Title() }

// LogItemDelegate is the delegate for LogItem.
type LogItemDelegate struct {
	common *common.Common
}

// Height returns the item height. Implements list.ItemDelegate.
func (d LogItemDelegate) Height() int { return 2 }

// Spacing returns the item spacing. Implements list.ItemDelegate.
func (d LogItemDelegate) Spacing() int { return 1 }

// Update updates the item. Implements list.ItemDelegate.
func (d LogItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	idx := m.Index()
	item, ok := m.SelectedItem().(LogItem)
	if !ok {
		return nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, d.common.KeyMap.Copy):
			item.copied = time.Now()
			d.common.Copy.Copy(item.Hash())
			return m.SetItem(idx, item)
		}
	}
	return nil
}

// Render renders the item. Implements list.ItemDelegate.
func (d LogItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(LogItem)
	if !ok {
		return
	}
	if i.Commit == nil {
		return
	}

	styles := d.common.Styles.LogItem.Normal
	if index == m.Index() {
		styles = d.common.Styles.LogItem.Active
	}

	horizontalFrameSize := styles.Base.GetHorizontalFrameSize()

	hash := i.Commit.ID.String()[:7]
	if !i.copied.IsZero() && i.copied.Add(time.Second).After(time.Now()) {
		hash = "copied"
	}
	title := styles.Title.Render(
		common.TruncateString(i.Title(),
			m.Width()-
				horizontalFrameSize-
				// 9 is the length of the hash (7) + the left padding (1) + the
				// title truncation symbol (1)
				9),
	)
	hashStyle := styles.Hash.Copy().
		Align(lipgloss.Right).
		PaddingLeft(1).
		Width(m.Width() -
			horizontalFrameSize -
			lipgloss.Width(title) - 1) // 1 is for the left padding
	if index == m.Index() {
		hashStyle = hashStyle.Bold(true)
	}
	hash = hashStyle.Render(hash)
	if m.Width()-horizontalFrameSize-hashStyle.GetHorizontalFrameSize()-hashStyle.GetWidth() <= 0 {
		hash = ""
		title = styles.Title.Render(
			common.TruncateString(i.Title(),
				m.Width()-horizontalFrameSize),
		)
	}
	author := i.Author.Name
	committer := i.Committer.Name
	who := ""
	if author != "" && committer != "" {
		who = styles.Keyword.Render(committer) + styles.Desc.Render(" committed")
		if author != committer {
			who = styles.Keyword.Render(author) + styles.Desc.Render(" authored and ") + who
		}
		who += " "
	}
	date := i.Committer.When.Format("Jan 02")
	if i.Committer.When.Year() != time.Now().Year() {
		date += fmt.Sprintf(" %d", i.Committer.When.Year())
	}
	who += styles.Desc.Render("on ") + styles.Keyword.Render(date)
	who = common.TruncateString(who, m.Width()-horizontalFrameSize)
	fmt.Fprint(w,
		styles.Base.Render(
			lipgloss.JoinVertical(lipgloss.Top,
				truncate.String(fmt.Sprintf("%s%s",
					title,
					hash,
				), uint(m.Width()-horizontalFrameSize)),
				who,
			),
		),
	)
}
