package yankable

import (
	"time"

	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Yankable struct {
	YankStyle lipgloss.Style
	Style     lipgloss.Style
	Text      string
	timer     timer.Model
	clicked   bool
}

func (y *Yankable) Init() tea.Cmd {
	y.timer = timer.New(3 * time.Second)
	return nil
}

func (y *Yankable) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case tea.MouseMsg:
		switch msg.Type {
		case tea.MouseRight:
			y.clicked = true
			cmds = append(cmds, y.timer.Init())
		}
	case timer.TimeoutMsg:
		y.clicked = false
	}
	return y, tea.Batch(cmds...)
}

func (y *Yankable) View() string {
	if y.clicked {
		return y.YankStyle.Render(y.Text)
	}
	return y.Style.Render(y.Text)
}
