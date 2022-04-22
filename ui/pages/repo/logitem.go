package repo

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/ui/styles"
	"github.com/muesli/reflow/truncate"
)

type LogItem struct {
	*git.Commit
}

func (i LogItem) ID() string {
	return i.Commit.ID.String()
}

func (i LogItem) Title() string {
	if i.Commit != nil {
		return strings.Split(i.Commit.Message, "\n")[0]
	}
	return ""
}

func (i LogItem) Description() string { return "" }

func (i LogItem) FilterValue() string { return i.Title() }

type LogItemDelegate struct {
	style *styles.Styles
}

func (d LogItemDelegate) Height() int                               { return 1 }
func (d LogItemDelegate) Spacing() int                              { return 0 }
func (d LogItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d LogItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(LogItem)
	if !ok {
		return
	}
	if i.Commit == nil {
		return
	}

	hash := i.Commit.ID.String()
	leftMargin := d.style.LogItemSelector.GetMarginLeft() +
		d.style.LogItemSelector.GetWidth() +
		d.style.LogItemHash.GetMarginLeft() +
		d.style.LogItemHash.GetWidth() +
		d.style.LogItemInactive.GetMarginLeft()
	title := truncateString(i.Title(), m.Width()-leftMargin, "…")
	if index == m.Index() {
		fmt.Fprint(w, d.style.LogItemSelector.Render(">")+
			d.style.LogItemHash.Bold(true).Render(hash[:7])+
			d.style.LogItemActive.Render(title))
	} else {
		fmt.Fprint(w, d.style.LogItemSelector.Render(" ")+
			d.style.LogItemHash.Render(hash[:7])+
			d.style.LogItemInactive.Render(title))
	}
}

func truncateString(s string, max int, tail string) string {
	if max < 0 {
		max = 0
	}
	return truncate.StringWithTail(s, uint(max), tail)
}
