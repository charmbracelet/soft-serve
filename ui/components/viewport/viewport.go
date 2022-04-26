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

// SetContent sets the viewport's content.
func (v *Viewport) SetContent(content string) {
	v.Viewport.SetContent(content)
}

// GotoTop moves the viewport to the top of the log.
func (v *Viewport) GotoTop() {
	v.Viewport.GotoTop()
}

// GotoBottom moves the viewport to the bottom of the log.
func (v *Viewport) GotoBottom() {
	v.Viewport.GotoBottom()
}

// HalfViewDown moves the viewport down by half the viewport height.
func (v *Viewport) HalfViewDown() {
	v.Viewport.HalfViewDown()
}

// HalfViewUp moves the viewport up by half the viewport height.
func (v *Viewport) HalfViewUp() {
	v.Viewport.HalfViewUp()
}

// ViewUp moves the viewport up by a page.
func (v *Viewport) ViewUp() []string {
	return v.Viewport.ViewUp()
}

// ViewDown moves the viewport down by a page.
func (v *Viewport) ViewDown() []string {
	return v.Viewport.ViewDown()
}

// LineUp moves the viewport up by the given number of lines.
func (v *Viewport) LineUp(n int) []string {
	return v.Viewport.LineUp(n)
}

// LineDown moves the viewport down by the given number of lines.
func (v *Viewport) LineDown(n int) []string {
	return v.Viewport.LineDown(n)
}

// ScrollPercent returns the viewport's scroll percentage.
func (v *Viewport) ScrollPercent() float64 {
	return v.Viewport.ScrollPercent()
}
