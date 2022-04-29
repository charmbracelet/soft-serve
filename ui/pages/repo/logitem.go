package repo

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/ui/styles"
	"github.com/muesli/reflow/truncate"
)

// LogItem is a item in the log list that displays a git commit.
type LogItem struct {
	*git.Commit
}

// ID implements selector.IdentifiableItem.
func (i LogItem) ID() string {
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
	style *styles.Styles
}

// Height returns the item height. Implements list.ItemDelegate.
func (d LogItemDelegate) Height() int { return 2 }

// Spacing returns the item spacing. Implements list.ItemDelegate.
func (d LogItemDelegate) Spacing() int { return 1 }

// Update updates the item. Implements list.ItemDelegate.
func (d LogItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

// Render renders the item. Implements list.ItemDelegate.
func (d LogItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(LogItem)
	if !ok {
		return
	}
	if i.Commit == nil {
		return
	}

	width := lipgloss.Width
	titleStyle := d.style.LogItemTitle.Copy()
	style := d.style.LogItemInactive
	if index == m.Index() {
		titleStyle.Bold(true)
		style = d.style.LogItemActive
	}
	hash := " " + i.Commit.ID.String()[:7]
	title := titleStyle.Render(
		truncateString(i.Title(), m.Width()-style.GetHorizontalFrameSize()-width(hash)-2, "…"),
	)
	hash = d.style.LogItemHash.Copy().
		Align(lipgloss.Right).
		Width(m.Width() -
			style.GetHorizontalFrameSize() -
			width(title) -
			// FIXME where this "1" is coming from?
			1).
		Render(hash)
	author := i.Author.Name
	commiter := i.Committer.Name
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
	who = truncateString(who, m.Width()-style.GetHorizontalFrameSize(), "…")
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

	// leftMargin := d.style.LogItemSelector.GetMarginLeft() +
	// 	d.style.LogItemSelector.GetWidth() +
	// 	d.style.LogItemHash.GetMarginLeft() +
	// 	d.style.LogItemHash.GetWidth() +
	// 	d.style.LogItemInactive.GetMarginLeft()
	// title := truncateString(i.Title(), m.Width()-leftMargin, "…")
	// if index == m.Index() {
	// 	fmt.Fprint(w, d.style.LogItemSelector.Render(">")+
	// 		d.style.LogItemHash.Bold(true).Render(hash[:7])+
	// 		d.style.LogItemActive.Render(title))
	// } else {
	// 	fmt.Fprint(w, d.style.LogItemSelector.Render(" ")+
	// 		d.style.LogItemHash.Render(hash[:7])+
	// 		d.style.LogItemInactive.Render(title))
	// }
	// fmt.Fprintln(w)
}

func truncateString(s string, max int, tail string) string {
	if max < 0 {
		max = 0
	}
	return truncate.StringWithTail(s, uint(max), tail)
}
