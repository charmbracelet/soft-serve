package tui

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type ViewportBubble struct {
	Viewport *viewport.Model
}

func (v *ViewportBubble) Init() tea.Cmd {
	return nil
}

func (v *ViewportBubble) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	vp, cmd := v.Viewport.Update(msg)
	v.Viewport = &vp
	return v, cmd
}

func (v *ViewportBubble) View() string {
	return v.Viewport.View()
}
