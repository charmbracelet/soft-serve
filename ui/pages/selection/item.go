package selection

import (
	"fmt"
	"io"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/soft-serve/proto"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/ui/common"
	"github.com/charmbracelet/soft-serve/ui/git"
	"github.com/dustin/go-humanize"
)

var _ sort.Interface = Items{}

// Items is a list of Item.
type Items []Item

// Len implements sort.Interface.
func (it Items) Len() int {
	return len(it)
}

// Less implements sort.Interface.
func (it Items) Less(i int, j int) bool {
	if it[i].lastUpdate == nil && it[j].lastUpdate != nil {
		return false
	}
	if it[i].lastUpdate != nil && it[j].lastUpdate == nil {
		return true
	}
	if it[i].lastUpdate == nil && it[j].lastUpdate == nil {
		return it[i].repo.Info.Name() < it[j].repo.Info.Name()
	}
	return it[i].lastUpdate.After(*it[j].lastUpdate)
}

// Swap implements sort.Interface.
func (it Items) Swap(i int, j int) {
	it[i], it[j] = it[j], it[i]
}

// Item represents a single item in the selector.
type Item struct {
	repo       git.Repository
	lastUpdate *time.Time
	cmd        string
	copied     time.Time
}

// New creates a new Item.
func NewItem(info proto.Metadata, cfg *config.Config) (Item, error) {
	repo, err := info.Open()
	if err != nil {
		log.Printf("error opening repo: %v", err)
		return Item{}, err
	}
	var lastUpdate *time.Time
	lu, err := repo.Repository().LatestCommitTime()
	if err == nil {
		lastUpdate = &lu
	}
	return Item{
		repo:       git.Repository{repo, info},
		lastUpdate: lastUpdate,
		cmd:        git.RepoURL(cfg.Host, cfg.SSH.Port, info.Name()),
	}, nil
}

// ID implements selector.IdentifiableItem.
func (i Item) ID() string {
	return i.repo.Info.Name()
}

// Title returns the item title. Implements list.DefaultItem.
func (i Item) Title() string {
	if pn := i.repo.Info.ProjectName(); pn != "" {
		return pn
	}
	return i.repo.Info.Name()
}

// Description returns the item description. Implements list.DefaultItem.
func (i Item) Description() string { return i.repo.Info.Description() }

// FilterValue implements list.Item.
func (i Item) FilterValue() string { return i.Title() }

// Command returns the item Command view.
func (i Item) Command() string {
	return i.cmd
}

// ItemDelegate is the delegate for the item.
type ItemDelegate struct {
	common     *common.Common
	activePane *pane
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
	i := listItem.(Item)
	s := strings.Builder{}
	var matchedRunes []int

	// Conditions
	var (
		isSelected = index == m.Index()
		isFiltered = m.FilterState() == list.Filtering || m.FilterState() == list.FilterApplied
	)

	styles := d.common.Styles.RepoSelector.Normal
	if isSelected {
		styles = d.common.Styles.RepoSelector.Active
	}

	title := i.Title()
	title = common.TruncateString(title, m.Width()-styles.Base.GetHorizontalFrameSize())
	if i.repo.Info.IsPrivate() {
		title += " 🔒"
	}
	if isSelected {
		title += " "
	}
	var updatedStr string
	if i.lastUpdate != nil {
		updatedStr = fmt.Sprintf(" Updated %s", humanize.Time(*i.lastUpdate))
	}
	if m.Width()-styles.Base.GetHorizontalFrameSize()-lipgloss.Width(updatedStr)-lipgloss.Width(title) <= 0 {
		updatedStr = ""
	}
	updatedStyle := styles.Updated.Copy().
		Align(lipgloss.Right).
		Width(m.Width() - styles.Base.GetHorizontalFrameSize() - lipgloss.Width(title))
	updated := updatedStyle.Render(updatedStr)

	if isFiltered && index < len(m.VisibleItems()) {
		// Get indices of matched characters
		matchedRunes = m.MatchesForItem(index)
	}

	if isFiltered {
		unmatched := styles.Title.Copy().Inline(true)
		matched := unmatched.Copy().Underline(true)
		title = lipgloss.StyleRunes(title, matchedRunes, matched, unmatched)
	}
	title = styles.Title.Render(title)
	desc := i.Description()
	desc = common.TruncateString(desc, m.Width()-styles.Base.GetHorizontalFrameSize())
	desc = styles.Desc.Render(desc)

	s.WriteString(lipgloss.JoinHorizontal(lipgloss.Bottom, title, updated))
	s.WriteRune('\n')
	s.WriteString(desc)
	s.WriteRune('\n')
	cmd := common.TruncateString(i.Command(), m.Width()-styles.Base.GetHorizontalFrameSize())
	cmd = styles.Command.Render(cmd)
	if !i.copied.IsZero() && i.copied.Add(time.Second).After(time.Now()) {
		cmd = styles.Command.Render("Copied!")
	}
	s.WriteString(cmd)
	fmt.Fprint(w,
		d.common.Zone.Mark(i.ID(),
			styles.Base.Render(s.String()),
		),
	)
}
