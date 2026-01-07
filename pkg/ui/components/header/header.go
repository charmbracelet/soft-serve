package header

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/soft-serve/pkg/ui/common"
)

// Header represents a header component.
type Header struct {
	common common.Common
	text   string
}

// New creates a new header component.
func New(c common.Common, text string) *Header {
	return &Header{
		common: c,
		text:   text,
	}
}

// SetSize implements common.Component.
func (h *Header) SetSize(width, height int) {
	h.common.SetSize(width, height)
}

// Init implements tea.Model.
func (h *Header) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (h *Header) Update(_ tea.Msg) (common.Model, tea.Cmd) {
	return h, nil
}

// View implements tea.Model.
func (h *Header) View() string {
	return h.common.Styles.ServerName.Render(strings.TrimSpace(h.text))
}
