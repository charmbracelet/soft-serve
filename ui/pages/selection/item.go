package selection

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/soft-serve/ui/components/yankable"
	"github.com/charmbracelet/soft-serve/ui/styles"
	"github.com/dustin/go-humanize"
)

// Item represents a single item in the selector.
type Item struct {
	Title       string
	Name        string
	Description string
	LastUpdate  time.Time
	URL         *yankable.Yankable
}

// ID implements selector.IdentifiableItem.
func (i Item) ID() string {
	return i.Name
}

// FilterValue implements list.Item.
func (i Item) FilterValue() string { return i.Title }

// ItemDelegate is the delegate for the item.
type ItemDelegate struct {
	styles    *styles.Styles
	activeBox *box
}

// Width returns the item width.
func (d ItemDelegate) Width() int {
	width := d.styles.MenuItem.GetHorizontalFrameSize() + d.styles.MenuItem.GetWidth()
	return width
}

// Height returns the item height. Implements list.ItemDelegate.
func (d ItemDelegate) Height() int {
	height := d.styles.MenuItem.GetVerticalFrameSize() + d.styles.MenuItem.GetHeight()
	return height
}

// Spacing returns the spacing between items. Implements list.ItemDelegate.
func (d ItemDelegate) Spacing() int { return 1 }

// Update implements list.ItemDelegate.
func (d ItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	cmds := make([]tea.Cmd, 0)
	if d.activeBox == nil || *d.activeBox != selectorBox {
		return nil
	}
	for i, item := range m.VisibleItems() {
		itm, ok := item.(Item)
		if !ok {
			continue
		}
		// FIXME check if X & Y are within the item box
		switch msg := msg.(type) {
		case tea.MouseMsg:
			x := msg.X
			y := msg.Y
			minX := (i * d.Width())
			maxX := minX + d.Width()
			minY := (i * d.Height())
			maxY := minY + d.Height()
			// log.Printf("i: %d, x: %d, y: %d", i, x, y)
			if y < minY || y > maxY || x < minX || x > maxX {
				continue
			}
		}
		y, cmd := itm.URL.Update(msg)
		itm.URL = y.(*yankable.Yankable)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

// Render implements list.ItemDelegate.
func (d ItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i := listItem.(Item)
	s := strings.Builder{}
	style := d.styles.MenuItem.Copy()
	if index == m.Index() {
		style = style.BorderForeground(d.styles.ActiveBorderColor)
		if d.activeBox != nil && *d.activeBox == readmeBox {
			// TODO make this into its own color
			style = style.BorderForeground(lipgloss.Color("15"))
		}
	}
	titleStr := i.Title
	updatedStr := fmt.Sprintf(" Updated %s", humanize.Time(i.LastUpdate))
	updated := d.styles.MenuLastUpdate.
		Copy().
		Width(m.Width() - style.GetHorizontalFrameSize() - lipgloss.Width(titleStr)).
		Render(updatedStr)
	title := lipgloss.NewStyle().
		Align(lipgloss.Left).
		Width(m.Width() - style.GetHorizontalFrameSize() - lipgloss.Width(updated)).
		Render(titleStr)

	s.WriteString(lipgloss.JoinHorizontal(lipgloss.Bottom, title, updated))
	s.WriteString("\n")
	s.WriteString(i.Description)
	s.WriteString("\n\n")
	s.WriteString(i.URL.View())
	w.Write([]byte(style.Render(s.String())))
}
