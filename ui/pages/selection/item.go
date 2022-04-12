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
func (d ItemDelegate) Spacing() int { return 1 }

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
		itemStyle = itemStyle.Copy().
			BorderStyle(lipgloss.Border{
				Left: "┃",
			}).
			BorderForeground(styles.ActiveBorderColor)
		if d.activeBox != nil && *d.activeBox == readmeBox {
			itemStyle = itemStyle.BorderForeground(styles.InactiveBorderColor)
		}
	}

	title := i.Title()
	title = common.TruncateString(title, m.Width()-itemStyle.GetHorizontalFrameSize())
	if i.repo.IsPrivate() {
		title += " 🔒"
	}
	if isSelected {
		title += " "
	}
	updatedStr := fmt.Sprintf(" Updated %s", humanize.Time(i.lastUpdate))
	if m.Width()-itemStyle.GetHorizontalFrameSize()-lipgloss.Width(updatedStr)-lipgloss.Width(title) <= 0 {
		updatedStr = ""
	}
	updatedStyle := styles.MenuLastUpdate.Copy().
		Align(lipgloss.Right).
		Width(m.Width() - itemStyle.GetHorizontalFrameSize() - lipgloss.Width(title))
	if isSelected {
		updatedStyle = updatedStyle.Bold(true)
	}
	updated := updatedStyle.Render(updatedStr)

	if isFiltered && index < len(m.VisibleItems()) {
		// Get indices of matched characters
		matchedRunes = m.MatchesForItem(index)
	}

	if isFiltered {
		unmatched := lipgloss.NewStyle().Inline(true)
		matched := unmatched.Copy().Underline(true)
		if isSelected {
			unmatched = unmatched.Bold(true)
			matched = matched.Bold(true)
		}
		title = lipgloss.StyleRunes(title, matchedRunes, matched, unmatched)
	}
	titleStyle := lipgloss.NewStyle()
	if isSelected {
		titleStyle = titleStyle.Bold(true)
	}
	title = titleStyle.Render(title)
	desc := i.Description()
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	if desc == "" {
		desc = "No description"
		descStyle = descStyle.Faint(true)
	}
	desc = common.TruncateString(desc, m.Width()-itemStyle.GetHorizontalFrameSize())
	desc = descStyle.Render(desc)

	s.WriteString(lipgloss.JoinHorizontal(lipgloss.Bottom, title, updated))
	s.WriteString("\n")
	s.WriteString(desc)
	s.WriteString("\n")
	cmdStyle := styles.RepoCommand.Copy()
	cmd := common.TruncateString(i.Command(), m.Width()-itemStyle.GetHorizontalFrameSize())
	cmd = cmdStyle.Render(cmd)
	if !i.copied.IsZero() && i.copied.Add(time.Second).After(time.Now()) {
		cmd = cmdStyle.Render("Copied!")
	}
	s.WriteString(cmd)
	fmt.Fprint(w, itemStyle.Render(s.String()))
}
