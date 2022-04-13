package yankable

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Yankable struct {
	yankStyle lipgloss.Style
	style     lipgloss.Style
	text      string
	clicked   bool
}

func New(style, yankStyle lipgloss.Style, text string) *Yankable {
	return &Yankable{
		yankStyle: yankStyle,
		style:     style,
		text:      text,
		clicked:   false,
	}
}

func (y *Yankable) SetText(text string) {
	y.text = text
}

func (y *Yankable) Init() tea.Cmd {
	return nil
}

func (y *Yankable) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case tea.MouseMsg:
		switch msg.Type {
		case tea.MouseRight:
			y.clicked = true
			return y, nil
		}
	default:
		y.clicked = false
	}
	return y, tea.Batch(cmds...)
}

func (y *Yankable) View() string {
	if y.clicked {
		return y.yankStyle.String()
	}
	return y.style.Render(y.text)
}
