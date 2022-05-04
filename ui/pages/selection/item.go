package selection

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/soft-serve/ui/common"
	"github.com/charmbracelet/soft-serve/ui/git"
	"github.com/dustin/go-humanize"
)

// Item represents a single item in the selector.
type Item struct {
	repo       git.GitRepo
	lastUpdate time.Time
	cmd        string
	copied     time.Time
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

// Command returns the item Command view.
func (i Item) Command() string {
	return i.cmd
}

// ItemDelegate is the delegate for the item.
type ItemDelegate struct {
	common    *common.Common
	activeBox *box
}

// Width returns the item width.
func (d ItemDelegate) Width() int {
	width := d.common.Styles.MenuItem.GetHorizontalFrameSize() + d.common.Styles.MenuItem.GetWidth()
	return width
}

// Height returns the item height. Implements list.ItemDelegate.
func (d ItemDelegate) Height() int {
	height := d.common.Styles.MenuItem.GetVerticalFrameSize() + d.common.Styles.MenuItem.GetHeight()
	return height
}

// Spacing returns the spacing between items. Implements list.ItemDelegate.
func (d ItemDelegate) Spacing() int { return 0 }

// Update implements list.ItemDelegate.
func (d ItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	idx := m.Index()
	item, ok := m.SelectedItem().(Item)
	if !ok {
		return nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, d.common.KeyMap.Copy):
			item.copied = time.Now()
			d.common.Copy.Copy(item.Command())
			return m.SetItem(idx, item)
		}
	}
	return nil
}

// Render implements list.ItemDelegate.
func (d ItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	styles := d.common.Styles
	i := listItem.(Item)
	s := strings.Builder{}
	var matchedRunes []int

	// Conditions
	var (
		isSelected = index == m.Index()
		isFiltered = m.FilterState() == list.Filtering || m.FilterState() == list.FilterApplied
	)

	itemStyle := styles.MenuItem.Copy()
	if isSelected {
		itemStyle = itemStyle.BorderForeground(styles.ActiveBorderColor)
		if d.activeBox != nil && *d.activeBox == readmeBox {
			// TODO make this into its own color
			itemStyle = itemStyle.BorderForeground(lipgloss.Color("15"))
		}
	}

	title := i.Title()
	updatedStr := fmt.Sprintf(" Updated %s", humanize.Time(i.lastUpdate))
	updated := styles.MenuLastUpdate.
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
	cmdStyle := styles.RepoCommand.Copy()
	cmd := cmdStyle.Render(i.Command())
	if !i.copied.IsZero() && i.copied.Add(time.Second).After(time.Now()) {
		cmd = cmdStyle.Render("Copied!")
	}
	s.WriteString(cmd)
	w.Write([]byte(itemStyle.Render(s.String())))
}
