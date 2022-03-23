package selection

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/soft-serve/internal/tui/style"
	"github.com/charmbracelet/soft-serve/pkg/tui/common"
	"github.com/muesli/reflow/truncate"
)

type SelectedMsg struct {
	Name  string
	Index int
}

type ActiveMsg struct {
	Name  string
	Index int
}

type Bubble struct {
	Items        []string
	SelectedItem int
	styles       *style.Styles
}

func NewBubble(items []string, styles *style.Styles) *Bubble {
	return &Bubble{
		Items:  items,
		styles: styles,
	}
}

func (b *Bubble) Init() tea.Cmd {
	return nil
}

func (b Bubble) View() string {
	s := strings.Builder{}
	repoNameMaxWidth := b.styles.Menu.GetWidth() - // menu width
		b.styles.Menu.GetHorizontalPadding() - // menu padding
		lipgloss.Width(b.styles.MenuCursor.String()) - // cursor
		b.styles.MenuItem.GetHorizontalFrameSize() // menu item gaps
	for i, item := range b.Items {
		item := truncate.StringWithTail(item, uint(repoNameMaxWidth), "…")
		if i == b.SelectedItem {
			s.WriteString(b.styles.MenuCursor.String())
			s.WriteString(b.styles.SelectedMenuItem.Render(item))
		} else {
			s.WriteString(b.styles.MenuItem.Render(item))
		}
		if i < len(b.Items)-1 {
			s.WriteRune('\n')
		}
	}
	return s.String()
}

func (b *Bubble) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "k", "up":
			if b.SelectedItem > 0 {
				b.SelectedItem--
				cmds = append(cmds, b.sendActiveMessage)
			}
		case "j", "down":
			if b.SelectedItem < len(b.Items)-1 {
				b.SelectedItem++
				cmds = append(cmds, b.sendActiveMessage)
			}
		case "enter":
			cmds = append(cmds, b.sendSelectedMessage)
		}
	}
	return b, tea.Batch(cmds...)
}

func (b *Bubble) Help() []common.HelpEntry {
	return []common.HelpEntry{
		{Key: "↑/↓", Value: "navigate"},
	}
}

func (b *Bubble) sendActiveMessage() tea.Msg {
	if b.SelectedItem >= 0 && b.SelectedItem < len(b.Items) {
		return ActiveMsg{
			Name:  b.Items[b.SelectedItem],
			Index: b.SelectedItem,
		}
	}
	return nil
}

func (b *Bubble) sendSelectedMessage() tea.Msg {
	if b.SelectedItem >= 0 && b.SelectedItem < len(b.Items) {
		return SelectedMsg{
			Name:  b.Items[b.SelectedItem],
			Index: b.SelectedItem,
		}
	}
	return nil
}
