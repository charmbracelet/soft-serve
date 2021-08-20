package selection

import (
	"smoothie/tui/style"

	tea "github.com/charmbracelet/bubbletea"
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
	s := ""
	for i, item := range b.Items {
		if i == b.SelectedItem {
			s += b.styles.MenuCursor.String()
			s += b.styles.SelectedMenuItem.Render(item)
		} else {
			s += b.styles.MenuItem.Render(item)
		}
		if i < len(b.Items)-1 {
			s += "\n"
		}
	}
	return s
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
