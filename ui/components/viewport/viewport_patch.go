package viewport

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// ViewportBubble represents a viewport component.
type ViewportBubble struct {
	Viewport *viewport.Model
}

// SetSize implements common.Component.
func (v *ViewportBubble) SetSize(width, height int) {
	v.Viewport.Width = width
	v.Viewport.Height = height
}

// Init implements tea.Model.
func (v *ViewportBubble) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (v *ViewportBubble) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	vp, cmd := v.Viewport.Update(msg)
	v.Viewport = &vp
	return v, cmd
}

// View implements tea.Model.
func (v *ViewportBubble) View() string {
	return v.Viewport.View()
}
