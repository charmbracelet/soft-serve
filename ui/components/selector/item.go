package selector

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/soft-serve/ui/components/yankable"
	"github.com/charmbracelet/soft-serve/ui/styles"
	"github.com/dustin/go-humanize"
)

type Item struct {
	Title       string
	Name        string
	Description string
	LastUpdate  time.Time
	URL         *yankable.Yankable
}

func (i *Item) Init() tea.Cmd {
	return nil
}

func (i *Item) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return i, nil
}

func (i *Item) View() string {
	return ""
}

func (i Item) FilterValue() string { return i.Title }

type ItemDelegate struct {
	styles *styles.Styles
}

func (d ItemDelegate) Width() int {
	width := d.styles.MenuItem.GetHorizontalFrameSize() + d.styles.MenuItem.GetWidth()
	return width
}
func (d ItemDelegate) Height() int {
	height := d.styles.MenuItem.GetVerticalFrameSize() + d.styles.MenuItem.GetHeight()
	return height
}
func (d ItemDelegate) Spacing() int { return 1 }
func (d ItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	cmds := make([]tea.Cmd, 0)
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
func (d ItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i := listItem.(Item)
	s := strings.Builder{}
	style := d.styles.MenuItem
	if index == m.Index() {
		style = d.styles.SelectedMenuItem
	}
	updated := d.styles.MenuLastUpdate.Render(fmt.Sprintf("Updated %s", humanize.Time(i.LastUpdate)))

	s.WriteString(fmt.Sprintf("%s %s", i.Title, updated))
	s.WriteString("\n")
	s.WriteString(i.Description)
	s.WriteString("\n\n")
	s.WriteString(i.URL.View())
	w.Write([]byte(style.Render(s.String())))
}
