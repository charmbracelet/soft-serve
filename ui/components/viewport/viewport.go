package viewport

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// Viewport represents a viewport component.
type Viewport struct {
	Viewport *viewport.Model
}

func New() *Viewport {
	return &Viewport{
		Viewport: &viewport.Model{
			MouseWheelEnabled: true,
		},
	}
}

// SetSize implements common.Component.
func (v *Viewport) SetSize(width, height int) {
	v.Viewport.Width = width
	v.Viewport.Height = height
}

// Init implements tea.Model.
func (v *Viewport) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (v *Viewport) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	vp, cmd := v.Viewport.Update(msg)
	v.Viewport = &vp
	return v, cmd
}

// View implements tea.Model.
func (v *Viewport) View() string {
	return v.Viewport.View()
}
