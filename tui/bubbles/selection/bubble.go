package selection

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	NormalStyle   lipgloss.Style
	SelectedStyle lipgloss.Style
	Items         []string
	selectedItem  int
}

func NewBubble(items []string) *Bubble {
	return &Bubble{
		NormalStyle:   normalStyle,
		SelectedStyle: selectedStyle,
		Items:         items,
	}
}

func (b *Bubble) Init() tea.Cmd {
	return nil
}

func (b *Bubble) View() string {
	s := ""
	for i, item := range b.Items {
		if i == b.selectedItem {
			s += b.SelectedStyle.Render(item) + "\n"
		} else {
			s += b.NormalStyle.Render(item) + "\n"
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
			if b.selectedItem > 0 {
				b.selectedItem--
				cmds = append(cmds, b.sendActiveMessage)
			}
		case "j", "down":
			if b.selectedItem < len(b.Items)-1 {
				b.selectedItem++
				cmds = append(cmds, b.sendActiveMessage)
			}
		case "enter":
			cmds = append(cmds, b.sendSelectedMessage)
		}
	}
	return b, tea.Batch(cmds...)
}

func (b *Bubble) sendActiveMessage() tea.Msg {
	if b.selectedItem >= 0 && b.selectedItem < len(b.Items) {
		return ActiveMsg{
			Name:  b.Items[b.selectedItem],
			Index: b.selectedItem,
		}
	}
	return nil
}

func (b *Bubble) sendSelectedMessage() tea.Msg {
	if b.selectedItem >= 0 && b.selectedItem < len(b.Items) {
		return SelectedMsg{
			Name:  b.Items[b.selectedItem],
			Index: b.selectedItem,
		}
	}
	return nil
}
