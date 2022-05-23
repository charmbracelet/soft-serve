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

var (
	highlight = lipgloss.NewStyle().Foreground(lipgloss.Color("#F1F1F1"))
)

// Render renders the item. Implements list.ItemDelegate.
func (d LogItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	styles := d.common.Styles
	i, ok := listItem.(LogItem)
	if !ok {
		return
	}
	if i.Commit == nil {
		return
	}

	width := lipgloss.Width
	titleStyle := styles.LogItemTitle.Copy()
	style := styles.LogItemInactive
	if index == m.Index() {
		titleStyle.Bold(true)
		style = styles.LogItemActive
	}
	hash := " " + i.Commit.ID.String()[:7]
	if !i.copied.IsZero() && i.copied.Add(time.Second).After(time.Now()) {
		hash = "copied"
	}
	title := titleStyle.Render(
		common.TruncateString(i.Title(), m.Width()-style.GetHorizontalFrameSize()-width(hash)-2),
	)
	hashStyle := styles.LogItemHash.Copy().
		Align(lipgloss.Right).
		Width(m.Width() -
			style.GetHorizontalFrameSize() -
			width(title) -
			// FIXME where this "1" is coming from?
			1)
	if index == m.Index() {
		hashStyle = hashStyle.Bold(true)
	}
	hash = hashStyle.Render(hash)
	author := highlight.Render(i.Author.Name)
	commiter := highlight.Render(i.Committer.Name)
	who := ""
	if author != "" && commiter != "" {
		who = fmt.Sprintf("%s committed", commiter)
		if author != commiter {
			who = fmt.Sprintf("%s authored and %s", author, who)
		}
		who += " "
	}
	date := fmt.Sprintf("on %s", i.Committer.When.Format("Feb 02"))
	if i.Committer.When.Year() != time.Now().Year() {
		date += fmt.Sprintf(" %d", i.Committer.When.Year())
	}
	who += date
	who = common.TruncateString(who, m.Width()-style.GetHorizontalFrameSize())
	fmt.Fprint(w,
		style.Render(
			lipgloss.JoinVertical(lipgloss.Top,
				lipgloss.JoinHorizontal(lipgloss.Left,
					title,
					hash,
				),
				who,
			),
		),
	)
}
