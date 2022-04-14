package header

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/soft-serve/ui/common"
)

// Header represents a header component.
type Header struct {
	common common.Common
	text   string
}

// New creates a new header component.
func New(c common.Common, text string) *Header {
	h := &Header{
		common: c,
		text:   text,
	}
	return h
}

// SetSize implements common.Component.
func (h *Header) SetSize(width, height int) {
	h.common.Width = width
	h.common.Height = height
}

// Init implements tea.Model.
func (h *Header) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (h *Header) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return h, nil
}

// View implements tea.Model.
func (h *Header) View() string {
	s := h.common.Styles.Header.Copy().Width(h.common.Width)
	return s.Render(strings.TrimSpace(h.text))
}
