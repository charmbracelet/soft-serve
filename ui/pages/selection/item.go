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
	"github.com/charmbracelet/soft-serve/ui/git"
	"github.com/charmbracelet/soft-serve/ui/styles"
	"github.com/dustin/go-humanize"
)

// Item represents a single item in the selector.
type Item struct {
	repo       git.GitRepo
	lastUpdate time.Time
	url        *yankable.Yankable
}

// ID implements selector.IdentifiableItem.
func (i Item) ID() string {
	return i.repo.Repo()
}

// Title returns the item title. Implements list.DefaultItem.
func (i Item) Title() string { return i.repo.Name() }

// Description returns the item description. Implements list.DefaultItem.
func (i Item) Description() string { return i.repo.Description() }

// FilterValue implements list.Item.
func (i Item) FilterValue() string { return i.Title() }

// URL returns the item URL view.
func (i Item) URL() string {
	return i.url.View()
}

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
func (d ItemDelegate) Spacing() int { return 0 }

// Update implements list.ItemDelegate.
func (d ItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	cmds := make([]tea.Cmd, 0)
	// if d.activeBox == nil || *d.activeBox != selectorBox {
	// 	return nil
	// }
	for i, item := range m.VisibleItems() {
		itm, ok := item.(Item)
		if !ok {
			continue
		}
		// FIXME check if X & Y are within the item box
		switch msg := msg.(type) {
		case tea.MouseMsg:
			// x := msg.X
			y := msg.Y
			// minX := (i * d.Width())
			// maxX := minX + d.Width()
			minY := (i * d.Height())
			maxY := minY + d.Height()
			// log.Printf("i: %d, x: %d, y: %d", i, x, y)
			if y < minY || y > maxY {
				continue
			}
		}
		y, cmd := itm.url.Update(msg)
		itm.url = y.(*yankable.Yankable)
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
	var matchedRunes []int

	// Conditions
	var (
		isSelected = index == m.Index()
		// emptyFilter = m.FilterState() == list.Filtering && m.FilterValue() == ""
		isFiltered = m.FilterState() == list.Filtering || m.FilterState() == list.FilterApplied
	)

	itemStyle := d.styles.MenuItem.Copy()
	if isSelected {
		itemStyle = itemStyle.BorderForeground(d.styles.ActiveBorderColor)
		if d.activeBox != nil && *d.activeBox == readmeBox {
			// TODO make this into its own color
			itemStyle = itemStyle.BorderForeground(lipgloss.Color("15"))
		}
	}

	title := i.Title()
	updatedStr := fmt.Sprintf(" Updated %s", humanize.Time(i.lastUpdate))
	updated := d.styles.MenuLastUpdate.
		Copy().
		Width(m.Width() - itemStyle.GetHorizontalFrameSize() - lipgloss.Width(title)).
		Render(updatedStr)
	titleStyle := lipgloss.NewStyle().
		Align(lipgloss.Left).
		Width(m.Width() - itemStyle.GetHorizontalFrameSize() - lipgloss.Width(updated))

	if isFiltered && index < len(m.VisibleItems()) {
		// Get indices of matched characters
		matchedRunes = m.MatchesForItem(index)
	}

	if isFiltered {
		unmatched := lipgloss.NewStyle().Inline(true)
		matched := unmatched.Copy().Underline(true)
		title = lipgloss.StyleRunes(title, matchedRunes, matched, unmatched)
	}
	title = titleStyle.Render(title)

	s.WriteString(lipgloss.JoinHorizontal(lipgloss.Bottom, title, updated))
	s.WriteString("\n")
	s.WriteString(i.Description())
	s.WriteString("\n\n")
	s.WriteString(i.url.View())
	w.Write([]byte(itemStyle.Render(s.String())))
}
