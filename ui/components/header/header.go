package header

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/soft-serve/ui/common"
)

type Header struct {
	common common.Common
	text   string
}

func New(c common.Common, text string) *Header {
	h := &Header{
		common: c,
		text:   text,
	}
	return h
}

func (h *Header) SetSize(width, height int) {
	h.common.Width = width
	h.common.Height = height
}

func (h *Header) Init() tea.Cmd {
	return nil
}

func (h *Header) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return h, nil
}

func (h *Header) View() string {
	s := h.common.Styles.Header.Copy().Width(h.common.Width)
	return s.Render(strings.TrimSpace(h.text))
}
