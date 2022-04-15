package yankable

import (
	"io"

	"github.com/aymanbagabas/go-osc52"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Yankable struct {
	yankStyle lipgloss.Style
	style     lipgloss.Style
	text      string
	clicked   bool
	osc52     *osc52.Output
}

func New(w io.Writer, environ []string, style, yankStyle lipgloss.Style, text string) *Yankable {
	return &Yankable{
		yankStyle: yankStyle,
		style:     style,
		text:      text,
		clicked:   false,
		osc52:     osc52.NewOutput(w, environ),
	}
}

func (y *Yankable) SetText(text string) {
	y.text = text
}

func (y *Yankable) Init() tea.Cmd {
	return nil
}

func (y *Yankable) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseMsg:
		switch msg.Type {
		case tea.MouseRight:
			y.clicked = true
			return y, y.copy()
		}
	default:
		y.clicked = false
	}
	return y, nil
}

func (y *Yankable) View() string {
	if y.clicked {
		return y.yankStyle.String()
	}
	return y.style.Render(y.text)
}

func (y *Yankable) copy() tea.Cmd {
	y.osc52.Copy(y.text)
	return nil
}
